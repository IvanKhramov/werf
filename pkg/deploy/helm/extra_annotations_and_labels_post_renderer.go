package helm

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	orig_yaml "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/werf/logboek"
	"github.com/werf/werf/pkg/werf"
)

var WerfRuntimeAnnotations = map[string]string{
	"werf.io/version": werf.Version,
}

var WerfRuntimeLabels = map[string]string{}

func NewExtraAnnotationsAndLabelsPostRenderer(extraAnnotations, extraLabels map[string]string) *ExtraAnnotationsAndLabelsPostRenderer {
	return &ExtraAnnotationsAndLabelsPostRenderer{
		ExtraAnnotations: extraAnnotations,
		ExtraLabels:      extraLabels,
	}
}

type ExtraAnnotationsAndLabelsPostRenderer struct {
	ExtraAnnotations map[string]string
	ExtraLabels      map[string]string
}

func findMapByKey(mapSlice orig_yaml.MapSlice, key string) orig_yaml.MapSlice {
	for _, item := range mapSlice {
		if itemKey, ok := item.Key.(string); ok {
			if itemKey == key {
				if itemValue, ok := item.Value.(orig_yaml.MapSlice); ok {
					return itemValue
				}
			}
		}
	}

	return nil
}

func setMapValueByKey(mapSlice orig_yaml.MapSlice, key string, value interface{}) (res orig_yaml.MapSlice) {
	var found bool
	for _, item := range mapSlice {
		if itemKey, ok := item.Key.(string); ok {
			if itemKey == key {
				res = append(res, orig_yaml.MapItem{Key: key, Value: value})
				found = true
				continue
			}
		}
		res = append(res, item)
	}

	if !found {
		res = append(res, orig_yaml.MapItem{Key: key, Value: value})
	}
	return
}

func (pr *ExtraAnnotationsAndLabelsPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	extraAnnotations := map[string]string{}
	for k, v := range WerfRuntimeAnnotations {
		extraAnnotations[k] = v
	}
	for k, v := range pr.ExtraAnnotations {
		extraAnnotations[k] = v
	}

	extraLabels := map[string]string{}
	for k, v := range WerfRuntimeLabels {
		extraLabels[k] = v
	}
	for k, v := range pr.ExtraLabels {
		extraLabels[k] = v
	}

	splitManifestsByKeys := releaseutil.SplitManifests(renderedManifests.String())

	manifestsKeys := make([]string, 0, len(splitManifestsByKeys))
	for k := range splitManifestsByKeys {
		manifestsKeys = append(manifestsKeys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(manifestsKeys))

	splitModifiedManifests := make([]string, 0)

	manifestNameRegex := regexp.MustCompile("# Source: .*")
	for _, manifestKey := range manifestsKeys {
		manifestContent := splitManifestsByKeys[manifestKey]
		manifestSource := manifestNameRegex.FindString(manifestContent)

		if os.Getenv("WERF_HELM_V3_EXTRA_ANNOTATIONS_AND_LABELS_DEBUG") == "1" {
			fmt.Printf("ExtraAnnotationsAndLabelsPostRenderer -- original manifest BEGIN\n")
			fmt.Printf("%s\n", manifestContent)
			fmt.Printf("ExtraAnnotationsAndLabelsPostRenderer -- original manifest END\n")
		}

		var obj unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(manifestContent), &obj); err != nil {
			logboek.Warn().LogF("Unable to decode yaml manifest as unstructured object: %s: will not add extra annotations and labels to this object:\n%s\n---\n", err, manifestContent)
			splitModifiedManifests = append(splitModifiedManifests, manifestContent)
			continue
		}
		if obj.GetKind() == "" {
			logboek.Debug().LogF("Skipping empty object\n")
			continue
		}

		var objMapSlice orig_yaml.MapSlice
		if err := orig_yaml.Unmarshal([]byte(manifestContent), &objMapSlice); err != nil {
			logboek.Warn().LogF("Unable to decode yaml manifest as map slice: %s: will not add extra annotations and labels to this object:\n%s\n---\n", err, manifestContent)
			splitModifiedManifests = append(splitModifiedManifests, manifestContent)
			continue
		}

		if os.Getenv("WERF_HELM_V3_EXTRA_ANNOTATIONS_AND_LABELS_DEBUG") == "1" {
			fmt.Printf("Unpacket obj annotations: %#v\n", obj.GetAnnotations())
		}

		if obj.IsList() && len(extraAnnotations) > 0 {
			logboek.Warn().LogF("werf annotations won't be applied to *List resource Kinds, including %s. We advise to replace *List resources with multiple separate resources of the same Kind\n", obj.GetKind())
		} else if len(extraAnnotations) > 0 {
			if metadata := findMapByKey(objMapSlice, "metadata"); metadata != nil {
				annotations := findMapByKey(metadata, "annotations")
				for k, v := range extraAnnotations {
					annotations = append(annotations, orig_yaml.MapItem{Key: k, Value: v})
				}
				metadata = setMapValueByKey(metadata, "annotations", annotations)
				objMapSlice = setMapValueByKey(objMapSlice, "metadata", metadata)
			}
		}

		if obj.IsList() && len(extraLabels) > 0 {
			logboek.Warn().LogF("werf labels won't be applied to *List resource Kinds, including %s. We advise to replace *List resources with multiple separate resources of the same Kind\n", obj.GetKind())
		} else if len(extraLabels) > 0 {
			if metadata := findMapByKey(objMapSlice, "metadata"); metadata != nil {
				labels := findMapByKey(metadata, "labels")
				for k, v := range extraLabels {
					labels = append(labels, orig_yaml.MapItem{Key: k, Value: v})
				}
				metadata = setMapValueByKey(metadata, "labels", labels)
				objMapSlice = setMapValueByKey(objMapSlice, "metadata", metadata)
			}
		}

		if modifiedManifestContent, err := orig_yaml.Marshal(objMapSlice); err != nil {
			return nil, fmt.Errorf("unable to modify manifest: %w\n%s\n---\n", err, manifestContent)
		} else {
			splitModifiedManifests = append(splitModifiedManifests, manifestSource+"\n"+string(modifiedManifestContent))

			if os.Getenv("WERF_HELM_V3_EXTRA_ANNOTATIONS_AND_LABELS_DEBUG") == "1" {
				fmt.Printf("ExtraAnnotationsAndLabelsPostRenderer -- modified manifest BEGIN\n")
				fmt.Printf("%s\n", modifiedManifestContent)
				fmt.Printf("ExtraAnnotationsAndLabelsPostRenderer -- modified manifest END\n")
			}
		}
	}

	modifiedManifests := bytes.NewBufferString(strings.Join(splitModifiedManifests, "\n---\n"))
	if os.Getenv("WERF_HELM_V3_EXTRA_ANNOTATIONS_AND_LABELS_DEBUG") == "1" {
		fmt.Printf("ExtraAnnotationsAndLabelsPostRenderer -- modified manifests RESULT BEGIN\n")
		fmt.Printf("%s\n", modifiedManifests.String())
		fmt.Printf("ExtraAnnotationsAndLabelsPostRenderer -- modified manifests RESULT END\n")
	}

	return modifiedManifests, nil
}

func (pr *ExtraAnnotationsAndLabelsPostRenderer) Add(extraAnnotations, extraLabels map[string]string) {
	if len(extraAnnotations) > 0 {
		if pr.ExtraAnnotations == nil {
			pr.ExtraAnnotations = make(map[string]string)
		}
		for k, v := range extraAnnotations {
			pr.ExtraAnnotations[k] = v
		}
	}

	if len(extraLabels) > 0 {
		if pr.ExtraLabels == nil {
			pr.ExtraLabels = make(map[string]string)
		}
		for k, v := range extraLabels {
			pr.ExtraLabels[k] = v
		}
	}
}
