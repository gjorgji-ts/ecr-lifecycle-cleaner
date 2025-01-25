// Copyright © 2025 Gjorgji J.

package cmd

import (
	"fmt"
	"log"
	"strings"

	initawsclient "ecr-lifecycle-cleaner/internal/initAwsClient"
	readpolicyfile "ecr-lifecycle-cleaner/internal/readPolicyFile"
	setlifecyclepolicy "ecr-lifecycle-cleaner/internal/setLifecyclePolicy"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var (
	policyFile string
)

// setPolicyCmd represents the setPolicy command
var setPolicyCmd = &cobra.Command{
	Use:   "setPolicy",
	Short: "Automates the management of lifecycle policies in ECR.",
	Long: `Automates the management of lifecycle policies in Amazon Elastic Container Registry (ECR).

Based on the provided policy, it sets lifecycle policies for specified repositories in the account.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[INFO] setPolicy called")

		if repoList != "" {
			repositoryList = strings.Split(repoList, ",")
		}

		policyText, err := readpolicyfile.ReadPolicyFile(policyFile)
		if err != nil {
			log.Fatalf("[ERROR] Reading policy file: %v", err)
		}

		client := initawsclient.InitAWSClient(config.LoadDefaultConfig)
		setlifecyclepolicy.Main(client, policyText, allRepos, repositoryList, repoPattern)
	},
}

func init() {
	rootCmd.AddCommand(setPolicyCmd)

	setPolicyCmd.Flags().StringVarP(&policyFile, "policyFile", "f", "", "Path to the JSON file containing the lifecycle policy")
	setPolicyCmd.MarkFlagRequired("policyFile")
}
