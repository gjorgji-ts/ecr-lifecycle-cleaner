// Copyright © 2024 Gjorgji J.

package initawsclient

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// InitAWSClient initializes the AWS client and returns it so it can be used in other functions.
func InitAWSClient() *ecr.Client {
	log.Println("============================================")
	log.Println("[INFO] Initializing AWS client...")

	cfg, err := config.LoadDefaultConfig(context.TODO())
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
