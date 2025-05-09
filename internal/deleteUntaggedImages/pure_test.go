// --- Copyright © 2025 Gjorgji J. ---

package deleteuntaggedimages

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
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
	got, err := ListImages(ctx, "repo", client)
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
	got, err := ListChildImages(ctx, "repo", []string{"d1"}, client)
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
	orphans, tagged, orphanCount, err := ImagesToDelete(ctx, "repo", client)
	if err != nil || tagged != 1 || orphanCount != 1 || !reflect.DeepEqual(orphans, []string{}) {
		t.Errorf("ImagesToDelete = %v, %d, %d, %v; want [], 1, 1, nil", orphans, tagged, orphanCount, err)
	}
}

func TestDeleteImages_Error(t *testing.T) {
	ctx := context.TODO()
	client := &mockECRClient{
		batchDeleteErr: errors.New("fail"),
	}
	_, _, err := DeleteImages(ctx, "repo", []string{"d1"}, client, false)
	if err == nil {
		t.Errorf("DeleteImages error case: want error, got nil")
	}
}
