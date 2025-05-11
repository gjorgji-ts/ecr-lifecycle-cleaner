// --- Copyright © 2025 Gjorgji J. ---

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

	// --- override the log output to avoid cluttering the test output ---
	log.SetOutput(io.Discard)

	client := ecr.NewFromConfig(cfg)

	// --- test with allRepos = true ---
	t.Run("Test with allRepos = true", func(t *testing.T) {
		err := Main(client, "mock-policy-text", true, nil, "", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// --- test with specific repository list ---
	t.Run("Test with specific repository list", func(t *testing.T) {
		err := Main(client, "mock-policy-text", false, []string{"test-repo"}, "", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// --- test with repository pattern ---
	t.Run("Test with repository pattern", func(t *testing.T) {
		err := Main(client, "mock-policy-text", false, nil, "test-.*", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// --- test with dryRun = true ---
	t.Run("Test with dryRun = true", func(t *testing.T) {
		err := Main(client, "mock-policy-text", true, nil, "", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

func TestSetPolicy(t *testing.T) {
	ctx := context.TODO()
	client := &ecr.Client{}
	// --- dry run should always succeed ---
	result, err := SetPolicy(ctx, client, "repo", "policy", true)
	if err != nil {
		t.Errorf("Expected no error in dry run, got: %v", err)
	}
	if result != "dry run: no changes made" {
		t.Errorf("Expected dry run message, got: %s", result)
	}
}

func TestSetPolicyForAll(t *testing.T) {
	ctx := context.TODO()
	client := &ecr.Client{}
	repos := []string{"repo1", "repo2"}
	results := SetPolicyForAll(ctx, client, "policy", repos, true)
	for repo, err := range results {
		if err != nil {
			t.Errorf("Expected no error for repo %s in dry run, got: %v", repo, err)
		}
	}
}
