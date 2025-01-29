// Copyright © 2025 Gjorgji J.

package cmd

import (
	"os"

	// "github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/spf13/cobra"
)

var (
	dryRun         bool
	allRepos       bool
	repoList       string
	repoPattern    string
	repositoryList []string
)

var managementGroup = &cobra.Group{
	ID:    "management",
	Title: "Management Commands:",
}

var rootCmd = &cobra.Command{
	Use:   "ecr-lifecycle-cleaner",
	Short: "A cli tool for managing ECR repositories.",
	Long: `A cli tool for managing ECR repositories.

It can be used to apply lifecycle policies to ECR repositories,
and clean up orphaned images from multi-platform builds.`,
}

// Execute runs the root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddGroup(managementGroup)

	cleanCmd.GroupID = managementGroup.ID
	setPolicyCmd.GroupID = managementGroup.ID

	rootCmd.PersistentFlags().BoolVarP(&allRepos, "allRepos", "a", false, "apply the changes to all repositories")
	rootCmd.PersistentFlags().StringVarP(&repoList, "repoList", "l", "", "comma-separated list of repository names (e.g., repo1,repo2)")
	rootCmd.PersistentFlags().StringVarP(&repoPattern, "repoPattern", "p", "", "regex pattern to match repository names (e.g., '^my-repo-.*'), make sure to quote the pattern to avoid shell interpretation")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dryRun", "d", false, "dry run mode, no changes will be applied")

	rootCmd.MarkFlagsOneRequired("allRepos", "repoList", "repoPattern")
	rootCmd.MarkFlagsMutuallyExclusive("allRepos", "repoList", "repoPattern")
}
