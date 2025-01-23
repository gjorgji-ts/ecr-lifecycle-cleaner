// Copyright © 2025 Gjorgji J.

package deleteuntaggedimages

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/smithy-go/middleware"
)

func TestDeleteUntaggedImages(t *testing.T) {
	// Mock middleware for DescribeRepositories
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

	// Mock middleware for ListImages
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

	// Mock middleware for BatchGetImage
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

	// Mock middleware for BatchDeleteImage
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

	// Override the log output to avoid cluttering the test output
	log.SetOutput(io.Discard)

	client := ecr.NewFromConfig(cfg)

	// Test with allRepos = true
	Main(client, true, nil, "")

	// Test with specific repository list
	Main(client, false, []string{"test-repo"}, "")

	// Test with repository pattern
	Main(client, false, nil, "test-.*")
}
