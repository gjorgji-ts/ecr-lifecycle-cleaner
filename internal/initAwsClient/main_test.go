// --- Copyright © 2025 Gjorgji J. ---

package initawsclient

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"
)

func TestInitAWSClient(t *testing.T) {
	getCallerIdentityMiddleware := middleware.FinalizeMiddlewareFunc(
		"GetCallerIdentityMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := middleware.GetOperationName(ctx)
			if operationName == "GetCallerIdentity" {
				return middleware.FinalizeOutput{
					Result: &sts.GetCallerIdentityOutput{
						Account: aws.String("123456789012"),
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	ecrMiddleware := middleware.FinalizeMiddlewareFunc(
		"ECRMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := middleware.GetOperationName(ctx)
			if operationName == "DescribeRepositories" {
				return middleware.FinalizeOutput{
					Result: &ecr.DescribeRepositoriesOutput{},
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
				if err := stack.Finalize.Add(getCallerIdentityMiddleware, middleware.Before); err != nil {
					return err
				}
				return stack.Finalize.Add(ecrMiddleware, middleware.Before)
			},
		}),
	)
	if err != nil {
		t.Fatalf("Unable to load SDK config: %v", err)
	}

	// --- override the log output to avoid cluttering the test output ---
	log.SetOutput(io.Discard)

	// --- use the mocked config to initialize the AWS client ---
	client := InitAWSClient(func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
		return cfg, nil
	})

	if client == nil {
		t.Fatalf("Expected non-nil ECR client")
	}
}

func TestNewECRClient(t *testing.T) {
	ctx := context.TODO()

	getCallerIdentityMiddleware := middleware.FinalizeMiddlewareFunc(
		"GetCallerIdentityMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := middleware.GetOperationName(ctx)
			if operationName == "GetCallerIdentity" {
				return middleware.FinalizeOutput{
					Result: &sts.GetCallerIdentityOutput{
						Account: aws.String("123456789012"),
					},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	ecrMiddleware := middleware.FinalizeMiddlewareFunc(
		"ECRMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			operationName := middleware.GetOperationName(ctx)
			if operationName == "DescribeRepositories" {
				return middleware.FinalizeOutput{
					Result: &ecr.DescribeRepositoriesOutput{},
				}, middleware.Metadata{}, nil
			}
			return handler.HandleFinalize(ctx, input)
		},
	)

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-west-2"),
		config.WithAPIOptions([]func(*middleware.Stack) error{
			func(stack *middleware.Stack) error {
				if err := stack.Finalize.Add(getCallerIdentityMiddleware, middleware.Before); err != nil {
					return err
				}
				return stack.Finalize.Add(ecrMiddleware, middleware.Before)
			},
		}),
	)
	if err != nil {
		t.Fatalf("Unable to load SDK config: %v", err)
	}

	mockLoader := func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
		return cfg, nil
	}

	client, _, region, err := NewECRClient(ctx, mockLoader)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Errorf("Expected non-nil ECR client")
	}
	if region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got: %s", region)
	}
	// --- account will be empty string in this mock, which is fine for this test ---
}
