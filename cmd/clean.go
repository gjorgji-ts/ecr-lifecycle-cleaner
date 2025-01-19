// Copyright © 2024 Gjorgji J.

package cmd

import (
	"fmt"

	"ecr-lifecycle-cleaner/internal/deleteUntaggedImages"
	"ecr-lifecycle-cleaner/internal/initAwsClient"

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
		client := initawsclient.InitAWSClient()
		deleteuntaggedimages.Main(client)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
