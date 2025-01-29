// Copyright © 2025 Gjorgji J.

package setlifecyclepolicy

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

func TestSetLifecyclePolicy(t *testing.T) {
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

	// Mock middleware for GetLifecyclePolicy
	getLifecyclePolicyMiddleware := middleware.FinalizeMiddlewareFunc(
		"GetLifecyclePolicyMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := awsMiddleware.GetOperationName(ctx)
			if operationName == "GetLifecyclePolicy" {
				return middleware.FinalizeOutput{
					Result: &ecr.GetLifecyclePolicyOutput{
						LifecyclePolicyText: aws.String("mock-policy-text"),
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	// Mock middleware for PutLifecyclePolicy
	putLifecyclePolicyMiddleware := middleware.FinalizeMiddlewareFunc(
		"PutLifecyclePolicyMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := awsMiddleware.GetOperationName(ctx)
			if operationName == "PutLifecyclePolicy" {
				return middleware.FinalizeOutput{
					Result: &ecr.PutLifecyclePolicyOutput{
						LifecyclePolicyText: aws.String("mock-policy-text"),
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
				if err := stack.Finalize.Add(getLifecyclePolicyMiddleware, middleware.Before); err != nil {
					return err
				}
				return stack.Finalize.Add(putLifecyclePolicyMiddleware, middleware.Before)
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
	t.Run("Test with allRepos = true", func(t *testing.T) {
		Main(client, "mock-policy-text", true, nil, "", false)
	})

	// Test with specific repository list
	t.Run("Test with specific repository list", func(t *testing.T) {
		Main(client, "mock-policy-text", false, []string{"test-repo"}, "", false)
	})

	// Test with repository pattern
	t.Run("Test with repository pattern", func(t *testing.T) {
		Main(client, "mock-policy-text", false, nil, "test-.*", false)
	})

	// Test with dryRun = true
	t.Run("Test with dryRun = true", func(t *testing.T) {
		Main(client, "mock-policy-text", true, nil, "", true)
	})
}
