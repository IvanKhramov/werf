package docker_registry

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/werf/werf/v2/pkg/image"
)

const AwsEcrImplementationName = "ecr"

var (
	awsEcrPatternRegexp = regexp.MustCompile(`^(\d{12})\.dkr\.ecr(-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.amazonaws\.com(\.cn)?$`)
	awsEcrPatterns      = []string{awsEcrPatternRegexp.String()}
)

type awsEcr struct {
	*defaultImplementation
}

type awsEcrOptions struct {
	defaultImplementationOptions
}

func newAwsEcr(options awsEcrOptions) (*awsEcr, error) {
	d, err := newDefaultAPIForImplementation(AwsEcrImplementationName, options.defaultImplementationOptions)
	if err != nil {
		return nil, err
	}

	awsEcr := &awsEcr{defaultImplementation: d}

	return awsEcr, nil
}

func (r *awsEcr) DeleteRepoImage(ctx context.Context, repoImage *image.Info) error {
	_, region, repository, err := r.parseReference(repoImage.Repository)
	if err != nil {
		return err
	}
	digest := repoImage.GetDigest()

	client, err := r.awsClient(ctx, region)
	if err != nil {
		return err
	}

	_, err = client.BatchDeleteImage(ctx, &ecr.BatchDeleteImageInput{
		ImageIds: []types.ImageIdentifier{
			{
				ImageDigest: &digest,
			},
		},
		RepositoryName: &repository,
	})

	return err
}

func (r *awsEcr) CreateRepo(ctx context.Context, reference string) error {
	_, region, repository, err := r.parseReference(reference)
	if err != nil {
		return err
	}

	client, err := r.awsClient(ctx, region)
	if err != nil {
		return err
	}

	if _, err := client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		ImageScanningConfiguration: nil,
		RepositoryName:             &repository,
		Tags:                       nil,
	}); err != nil {
		return err
	}

	return nil
}

func (r *awsEcr) DeleteRepo(ctx context.Context, reference string) error {
	_, region, repository, err := r.parseReference(reference)
	if err != nil {
		return err
	}

	client, err := r.awsClient(ctx, region)
	if err != nil {
		return err
	}

	if _, err := client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		Force:          true,
		RegistryId:     nil,
		RepositoryName: &repository,
	}); err != nil {
		return err
	}

	return nil
}

func (r *awsEcr) awsClient(ctx context.Context, region string) (*ecr.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws configuration: %w", err)
	}

	client := ecr.NewFromConfig(cfg)
	return client, nil
}

func (r *awsEcr) String() string {
	return AwsEcrImplementationName
}

func (r *awsEcr) parseReference(reference string) (string, string, string, error) {
	var registryId, region, repository string

	parsedReference, err := name.NewRepository(reference)
	if err != nil {
		return "", "", "", err
	}

	registryId, region, err = r.parseHostname(parsedReference.RegistryStr())
	if err != nil {
		return "", "", "", err
	}

	repository = parsedReference.RepositoryStr()

	return registryId, region, repository, nil
}

func (r *awsEcr) parseHostname(hostname string) (string, string, error) {
	var registryId, region string

	splitURL := awsEcrPatternRegexp.FindStringSubmatch(hostname)
	if len(splitURL) == 0 {
		return "", "", fmt.Errorf("%s is not a valid ECR repository URL", hostname)
	}

	registryId = splitURL[1]
	region = splitURL[3]

	return registryId, region, nil
}
