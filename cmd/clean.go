// Copyright © 2025 Gjorgji J.

package cmd

import (
	"fmt"
	"strings"

	initawsclient "ecr-lifecycle-cleaner/internal/initAwsClient"
	deleteuntaggedimages "ecr-lifecycle-cleaner/internal/deleteUntaggedImages"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Automates the cleanup of untagged images in ECR.",
	Long: `Automates the cleanup of untagged images in Amazon Elastic Container Registry (ECR).

It retrieves all repositories, identifies untagged images that are not referenced by any tagged images,
and deletes those untagged images to help manage storage and maintain a clean registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[INFO] clean called")

		if repoList != "" {
			repositoryList = strings.Split(repoList, ",")
		}

		client := initawsclient.InitAWSClient(config.LoadDefaultConfig)
		deleteuntaggedimages.Main(client, allRepos, repositoryList, repoPattern)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
