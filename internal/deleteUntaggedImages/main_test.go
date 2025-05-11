// --- Copyright © 2025 Gjorgji J. ---

package deleteuntaggedimages

import (
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/smithy-go/middleware"
)

// --- mock ECR client ---
type mockECRClient struct {
	ecr.Client
	describeReposOut *ecr.DescribeRepositoriesOutput
	describeReposErr error
	listImagesOut    *ecr.ListImagesOutput
	listImagesErr    error
	batchGetOut      *ecr.BatchGetImageOutput
	batchGetErr      error
	batchDeleteOut   *ecr.BatchDeleteImageOutput
	batchDeleteErr   error
}

func (m *mockECRClient) DescribeRepositories(ctx context.Context, in *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	return m.describeReposOut, m.describeReposErr
}
func (m *mockECRClient) ListImages(ctx context.Context, in *ecr.ListImagesInput, optFns ...func(*ecr.Options)) (*ecr.ListImagesOutput, error) {
	return m.listImagesOut, m.listImagesErr
}
func (m *mockECRClient) BatchGetImage(ctx context.Context, in *ecr.BatchGetImageInput, optFns ...func(*ecr.Options)) (*ecr.BatchGetImageOutput, error) {
	return m.batchGetOut, m.batchGetErr
}
func (m *mockECRClient) BatchDeleteImage(ctx context.Context, in *ecr.BatchDeleteImageInput, optFns ...func(*ecr.Options)) (*ecr.BatchDeleteImageOutput, error) {
	return m.batchDeleteOut, m.batchDeleteErr
}

func TestDeleteUntaggedImages(t *testing.T) {
	describeRepositoriesMiddleware := middleware.FinalizeMiddlewareFunc(
		"DescribeRepositoriesMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := awsMiddleware.GetOperationName(ctx)
			if operationName == "DescribeRepositories" {
				return middleware.FinalizeOutput{
					Result: &ecr.DescribeRepositoriesOutput{
						Repositories: []types.Repository{
							{RepositoryName: aws.String("test-repo")},
						},
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	listImagesMiddleware := middleware.FinalizeMiddlewareFunc(
		"ListImagesMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := awsMiddleware.GetOperationName(ctx)
			if operationName == "ListImages" {
				return middleware.FinalizeOutput{
					Result: &ecr.ListImagesOutput{
						ImageIds: []types.ImageIdentifier{
							{ImageDigest: aws.String("sha256:1234"), ImageTag: aws.String("latest")},
							{ImageDigest: aws.String("sha256:5678")},
						},
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	batchGetImageMiddleware := middleware.FinalizeMiddlewareFunc(
		"BatchGetImageMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := awsMiddleware.GetOperationName(ctx)
			if operationName == "BatchGetImage" {
				return middleware.FinalizeOutput{
					Result: &ecr.BatchGetImageOutput{
						Images: []types.Image{
							{
								ImageManifest: aws.String(`{"manifests":[{"digest":"sha256:5678"}]}`),
							},
						},
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	batchDeleteImageMiddleware := middleware.FinalizeMiddlewareFunc(
		"BatchDeleteImageMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := awsMiddleware.GetOperationName(ctx)
			if operationName == "BatchDeleteImage" {
				return middleware.FinalizeOutput{
					Result: &ecr.BatchDeleteImageOutput{
						ImageIds: []types.ImageIdentifier{
							{ImageDigest: aws.String("sha256:5678")},
						},
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("us-west-2"),
		config.WithAPIOptions([]func(*middleware.Stack) error{
			func(stack *middleware.Stack) error {
				if err := stack.Finalize.Add(describeRepositoriesMiddleware, middleware.Before); err != nil {
					return err
				}
				if err := stack.Finalize.Add(listImagesMiddleware, middleware.Before); err != nil {
					return err
				}
				if err := stack.Finalize.Add(batchGetImageMiddleware, middleware.Before); err != nil {
					return err
				}
				return stack.Finalize.Add(batchDeleteImageMiddleware, middleware.Before)
			},
		}),
	)
	if err != nil {
		t.Fatalf("Unable to load SDK config: %v", err)
	}

	// --- override the log output to avoid cluttering the test output ---
	log.SetOutput(io.Discard)

	client := ecr.NewFromConfig(cfg)

	// --- test with allRepos = true ---
	t.Run("Test with allRepos = true", func(t *testing.T) {
		err := Main(client, true, nil, "", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// --- test with specific repository list ---
	t.Run("Test with specific repository list", func(t *testing.T) {
		err := Main(client, false, []string{"test-repo"}, "", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// --- test with repository pattern ---
	t.Run("Test with repository pattern", func(t *testing.T) {
		err := Main(client, false, nil, "test-.*", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// --- test with dryRun = true ---
	t.Run("Test with dryRun = true", func(t *testing.T) {
		err := Main(client, true, nil, "", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

func TestDeleteImages_DryRun(t *testing.T) {
	client := &ecr.Client{} // --- not making real calls in this test ---
	ctx := context.TODO()
	deleted, failed, err := deleteImages(ctx, "fake-repo", []string{"sha256:deadbeef"}, client, true)
	if err != nil {
		t.Errorf("Expected no error in dry run, got: %v", err)
	}
	if deleted != 0 || failed != 0 {
		t.Errorf("Expected 0 deleted/failed in dry run, got: %d/%d", deleted, failed)
	}
}

func TestListRepositories(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		describeReposOut: &ecr.DescribeRepositoriesOutput{
			Repositories: []types.Repository{{RepositoryName: aws.String("repo1")}},
		},
	}
	got, err := ListRepositories(ctx, client)
	if err != nil || !reflect.DeepEqual(got, []string{"repo1"}) {
		t.Errorf("ListRepositories = %v, %v; want [repo1], nil", got, err)
	}
}

func TestListRepositoriesByPattern(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		describeReposOut: &ecr.DescribeRepositoriesOutput{
			Repositories: []types.Repository{{RepositoryName: aws.String("foo")}, {RepositoryName: aws.String("bar")}},
		},
	}
	got, err := ListRepositoriesByPattern(ctx, client, "^f")
	if err != nil || !reflect.DeepEqual(got, []string{"foo"}) {
		t.Errorf("ListRepositoriesByPattern = %v, %v; want [foo], nil", got, err)
	}
}

func TestListImages(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		listImagesOut: &ecr.ListImagesOutput{
			ImageIds: []types.ImageIdentifier{
				{ImageDigest: aws.String("d1"), ImageTag: aws.String("t1")},
				{ImageDigest: aws.String("d2")},
			},
		},
	}
	got, err := listImages(ctx, "repo", client)
	want := map[string][]string{"tagged": {"d1"}, "orphan": {"d2"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("ListImages = %v, %v; want %v, nil", got, err, want)
	}
}

func TestListChildImages(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		batchGetOut: &ecr.BatchGetImageOutput{
			Images: []types.Image{{ImageManifest: aws.String(`{"manifests":[{"digest":"d2"}]}`)}},
		},
	}
	got, err := listChildImages(ctx, "repo", []string{"d1"}, client)
	if err != nil || !reflect.DeepEqual(got, []string{"d2"}) {
		t.Errorf("ListChildImages = %v, %v; want [d2], nil", got, err)
	}
}

func TestImagesToDelete(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		listImagesOut: &ecr.ListImagesOutput{
			ImageIds: []types.ImageIdentifier{
				{ImageDigest: aws.String("d1"), ImageTag: aws.String("t1")},
				{ImageDigest: aws.String("d2")},
			},
		},
		batchGetOut: &ecr.BatchGetImageOutput{
			Images: []types.Image{{ImageManifest: aws.String(`{"manifests":[{"digest":"d2"}]}`)}},
		},
	}
	orphans, tagged, orphanCount, err := imagesToDelete(ctx, "repo", client)
	if err != nil || tagged != 1 || orphanCount != 1 || !reflect.DeepEqual(orphans, []string{}) {
		t.Errorf("ImagesToDelete = %v, %d, %d, %v; want [], 1, 1, nil", orphans, tagged, orphanCount, err)
	}
}

func TestDeleteImages_Error(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		batchDeleteErr: errors.New("fail"),
	}
	_, _, err := deleteImages(ctx, "repo", []string{"d1"}, client, false)
	if err == nil {
		t.Errorf("DeleteImages error case: want error, got nil")
	}
}

func TestImagesToDeleteWithLogging(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		listImagesOut: &ecr.ListImagesOutput{
			ImageIds: []types.ImageIdentifier{},
		},
	}
	var logMessages []string
	var mu sync.Mutex
	orphans, err := imagesToDeleteWithLogging(ctx, "repo", client, &logMessages, &mu)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if orphans == nil {
		t.Errorf("Expected empty slice, got nil")
	}
}

func TestDeleteImagesWithLogging_DryRun(t *testing.T) {
	ctx := context.TODO()
	client := &ecr.Client{}
	var logMessages []string
	var mu sync.Mutex
	err := deleteImagesWithLogging(ctx, "repo", []string{"sha256:deadbeef"}, client, true, &logMessages, &mu)
	if err != nil {
		t.Errorf("Expected no error in dry run, got: %v", err)
	}
}

func TestCleanECRWithLogging_EmptyRepos(t *testing.T) {
	ctx := context.TODO()
	client := &ecr.Client{}
	err := CleanECRWithLogging(ctx, client, []string{}, true)
	if err != nil {
		t.Errorf("Expected no error for empty repo list, got: %v", err)
	}
}

func TestFilterOrphans(t *testing.T) {
	orphans := []string{"a", "b", "c", "d"}
	children := []string{"b", "d"}
	filtered := filterOrphans(orphans, children)
	want := []string{"a", "c"}
	if !reflect.DeepEqual(filtered, want) {
		t.Errorf("filterOrphans = %v; want %v", filtered, want)
	}
}
