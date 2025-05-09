// --- Copyright © 2025 Gjorgji J. ---

package initawsclient

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// --- loads AWS configuration ---
type ConfigLoader func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error)

// --- initializes the AWS client and returns it so it can be used in other functions ---
func InitAWSClient(loadConfig ConfigLoader) *ecr.Client {
	log.Println("============================================")
	log.Println("[INFO] Initializing AWS client...")

	cfg, err := loadConfig(context.TODO())
	if err != nil {
		log.Fatalf("[ERROR] Unable to load SDK config: %v", err)
	}
	log.Println("[INFO] AWS SDK config loaded successfully")

	stsClient := sts.NewFromConfig(cfg)
	log.Println("[INFO] STS client created successfully")

	identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatalf("[ERROR] Unable to get caller identity: %v", err)
	}
	log.Printf("[INFO] Using AWS account ID: %s", aws.ToString(identity.Account))
	log.Printf("[INFO] Using AWS region: %s", cfg.Region)

	client := ecr.NewFromConfig(cfg)
	log.Println("[INFO] ECR client created successfully")
	log.Println("============================================")

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
