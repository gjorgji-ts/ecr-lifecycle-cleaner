// Copyright © 2024 Gjorgji J.

package cmd

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/spf13/cobra"
)

var (
	managementGroup = &cobra.Group{
		ID:    "management",
		Title: "Management Commands:",
	}
)

var client *ecr.Client

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ecr-lifecycle-cleaner",
	Short: "A cli tool for managing ECR repositories.",
	Long: `A cli tool for managing ECR repositories.

It can be used to apply lifecycle policies to ECR repositories,
and clean up orphaned images from multi-platform builds.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add command groups to the root command
	rootCmd.AddGroup(managementGroup)

	// Assign commands to groups
	cleanCmd.GroupID = managementGroup.ID
	setPolicyCmd.GroupID = managementGroup.ID
}
