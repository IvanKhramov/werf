package e2e_export_test

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/werf/v2/test/pkg/utils"
	"github.com/werf/werf/v2/test/pkg/werf"
)

type simpleTestOptions struct {
	Platforms    []string
	CustomLabels []string
}

var _ = Describe("Simple export", Label("e2e", "export", "simple"), func() {
	DescribeTable("should succeed and export images",
		func(opts simpleTestOptions) {
			By("initializating")
			{
				SuiteData.WerfRepo = strings.Join([]string{SuiteData.RegistryLocalAddress, SuiteData.ProjectName}, "/")
				SuiteData.Stubs.SetEnv("WERF_REPO", SuiteData.WerfRepo)
				SuiteData.Stubs.SetEnv("DOCKER_BUILDKIT", "1")
				repoDirname := "repo"
				fixtureRelPath := "simple"

				By("preparing test repo")
				SuiteData.InitTestRepo(repoDirname, fixtureRelPath)

				By("running export")
				werfProject := werf.NewProject(SuiteData.WerfBinPath, SuiteData.GetTestRepoPath(repoDirname))
				imageName := fmt.Sprintf("%s/werf-export-%s", SuiteData.RegistryLocalAddress, utils.GetRandomString(10))

				exportArgs := []string{
					"--tag",
					imageName,
				}
				if len(opts.Platforms) > 0 {
					for _, platform := range opts.Platforms {
						exportArgs = append(exportArgs, "--platform", platform)
					}
				}
				if len(opts.CustomLabels) > 0 {
					for _, label := range opts.CustomLabels {
						exportArgs = append(exportArgs, "--add-label", label)
					}
				}

				exportOut := werfProject.Export(&werf.ExportOptions{
					CommonOptions: werf.CommonOptions{
						ExtraArgs: exportArgs,
					},
				})
				Expect(exportOut).To(ContainSubstring("Exporting image..."))

				By("checking result")
				commonCheckImageConfigFunc := func(config v1.Config) {
					for key := range config.Labels {
						if strings.HasSuffix(key, "werf") {
							Fail(fmt.Sprintf("labels %s should not contain werf service labels", config.Labels))
						}
					}
				}

				var checkImageConfigFunc func(v1.Config)
				var checkIndexManifestFunc func(*v1.IndexManifest)
				if len(opts.Platforms) > 0 {
					checkIndexManifestFunc = func(manifest *v1.IndexManifest) {
					platformLoop:
						for _, platform := range opts.Platforms {
							for _, i := range manifest.Manifests {
								if i.Platform.String() == platform {
									break platformLoop
								}
							}

							Fail(fmt.Sprintf("platform %s not found in index manifest", platform))
						}

						Expect(len(opts.Platforms)).To(Equal(len(manifest.Manifests)), "unexpected number of platforms in index manifest")
					}
				}
				if len(opts.CustomLabels) > 0 {
					checkImageConfigFunc = func(config v1.Config) {
						for _, label := range opts.CustomLabels {
							labelParts := strings.Split(label, "=")
							Expect(config.Labels).To(HaveKeyWithValue(labelParts[0], labelParts[1]))
						}

						commonCheckImageConfigFunc(config)
					}
				} else {
					checkImageConfigFunc = commonCheckImageConfigFunc
				}

				if len(opts.Platforms) > 0 {
					checkIndexManifest(imageName, checkIndexManifestFunc, checkImageConfigFunc)
				} else {
					checkImageManifest(imageName, checkImageConfigFunc)
				}
			}
		},
		Entry("base", simpleTestOptions{
			CustomLabels: []string{
				"TEST_LABEL=TEST_VALUE",
			},
		}),
		Entry("multiplatform", simpleTestOptions{
			CustomLabels: []string{
				"TEST_LABEL=TEST_VALUE",
			},
			Platforms: []string{
				"linux/amd64",
				"linux/arm64",
			},
		}),
	)
})

func checkIndexManifest(reference string, checkIndexManifestFunc func(*v1.IndexManifest), checkImageConfigFunc func(v1.Config)) {
	ref, err := name.ParseReference(reference)
	Ω(err).ShouldNot(HaveOccurred(), err)

	desc, err := remote.Get(ref)
	Ω(err).ShouldNot(HaveOccurred(), err)

	Ω(desc.MediaType.IsIndex()).Should(BeTrue(), "expected index, got image")

	ii, err := desc.ImageIndex()
	Ω(err).ShouldNot(HaveOccurred(), err)

	im, err := ii.IndexManifest()
	Ω(err).ShouldNot(HaveOccurred(), err)

	checkIndexManifestFunc(im)

	for _, m := range im.Manifests {
		subref := fmt.Sprintf("%s@%s", reference, m.Digest)
		checkImageManifest(subref, checkImageConfigFunc)
	}
}

func checkImageManifest(reference string, checkImageConfigFunc func(v1.Config)) {
	ref, err := name.ParseReference(reference)
	Ω(err).ShouldNot(HaveOccurred(), err)

	desc, err := remote.Get(ref)
	Ω(err).ShouldNot(HaveOccurred(), err)

	Ω(desc.MediaType.IsIndex()).ShouldNot(BeTrue(), "expected image, got index")

	i, err := desc.Image()
	Ω(err).ShouldNot(HaveOccurred(), err)

	c, err := i.ConfigFile()
	Ω(err).ShouldNot(HaveOccurred(), err)

	checkImageConfigFunc(c.Config)
}
