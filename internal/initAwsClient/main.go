// --- Copyright © 2025 Gjorgji J. ---

package initawsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// --- loads AWS configuration ---
type ConfigLoader func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error)

// --- initializes the AWS client and returns it so it can be used in other functions ---
func InitAWSClient(loadConfig ConfigLoader) *ecr.Client {
	cfg, err := loadConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	stsClient := sts.NewFromConfig(cfg)

	_, err = stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}

	client := ecr.NewFromConfig(cfg)

	return client
}

// --- returns ECR client and account info, no logging or side effects ---
func NewECRClient(ctx context.Context, loadConfig ConfigLoader) (*ecr.Client, string, string, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, "", "", err
	}
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, "", "", err
	}
	client := ecr.NewFromConfig(cfg)
	return client, aws.ToString(identity.Account), cfg.Region, nil
}
