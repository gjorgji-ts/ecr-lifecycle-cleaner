// Copyright © 2024 Gjorgji J.

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
	// Mock middleware for GetCallerIdentity
	getCallerIdentityMiddleware := middleware.FinalizeMiddlewareFunc(
		"GetCallerIdentityMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			return middleware.FinalizeOutput{
				Result: &sts.GetCallerIdentityOutput{
					Account: aws.String("123456789012"),
				},
			}, middleware.Metadata{}, nil
		},
	)

	// Mock middleware for ECR client creation (no specific output needed)
	ecrMiddleware := middleware.FinalizeMiddlewareFunc(
		"ECRMock",
		func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			return middleware.FinalizeOutput{
				Result: &ecr.DescribeRepositoriesOutput{},
			}, middleware.Metadata{}, nil
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

	// Override the log output to avoid cluttering the test output
	log.SetOutput(io.Discard)

	// Use the mocked config to initialize the AWS client
	client := ecr.NewFromConfig(cfg)

	if client == nil {
		t.Fatalf("Expected non-nil ECR client")
	}
}
