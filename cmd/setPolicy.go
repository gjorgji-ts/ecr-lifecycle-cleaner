// Copyright © 2024 Gjorgji J.

package cmd

import (
	"fmt"
	"log"
	"strings"

	"ecr-lifecycle-cleaner/internal/initAwsClient"
	"ecr-lifecycle-cleaner/internal/readPolicyFile"
	"ecr-lifecycle-cleaner/internal/setLifecyclePolicy"

	"github.com/spf13/cobra"
)

var (
	policyFile     string
	allRepos       bool
	repoList       string
	repoPattern    string
	repositoryList []string
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

		client := initawsclient.InitAWSClient()
		setlifecyclepolicy.Main(client, policyText, allRepos, repositoryList, repoPattern)
	},
}

func init() {
	rootCmd.AddCommand(setPolicyCmd)

	setPolicyCmd.Flags().StringVarP(&policyFile, "policy-file", "p", "", "Path to the JSON file containing the lifecycle policy")
	setPolicyCmd.Flags().BoolVar(&allRepos, "allRepos", false, "Apply the policy to all repositories")
	setPolicyCmd.Flags().StringVar(&repoList, "repoList", "", "Comma-separated list of repository names to apply the policy to (e.g., repo1,repo2)")
	setPolicyCmd.Flags().StringVar(&repoPattern, "repoPattern", "", "Regex pattern to match repository names to (e.g., ^my-repo-.*$)")
	setPolicyCmd.MarkFlagRequired("policy-file")
	setPolicyCmd.MarkFlagsOneRequired("allRepos", "repoList", "repoPattern")
	setPolicyCmd.MarkFlagsMutuallyExclusive("allRepos", "repoList", "repoPattern")
}
