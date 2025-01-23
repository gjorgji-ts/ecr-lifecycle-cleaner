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

var (
	allRepos       bool
	repoList       string
	repoPattern    string
	repositoryList []string
)

var rootCmd = &cobra.Command{
	Use:   "ecr-lifecycle-cleaner",
	Short: "A cli tool for managing ECR repositories.",
	Long: `A cli tool for managing ECR repositories.

It can be used to apply lifecycle policies to ECR repositories,
and clean up orphaned images from multi-platform builds.`,
}

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

	rootCmd.PersistentFlags().BoolVarP(&allRepos, "allRepos", "a", false, "Apply the changes to all repositories")
	rootCmd.PersistentFlags().StringVarP(&repoList, "repoList", "l", "", "Comma-separated list of repository names (e.g., repo1,repo2)")
	rootCmd.PersistentFlags().StringVarP(&repoPattern, "repoPattern", "p", "", "Regex pattern to match repository names (e.g., '^my-repo-.*'). Make sure to quote the pattern to avoid shell interpretation.")

	rootCmd.MarkFlagsOneRequired("allRepos", "repoList", "repoPattern")
	rootCmd.MarkFlagsMutuallyExclusive("allRepos", "repoList", "repoPattern")
}
