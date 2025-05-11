// --- Copyright © 2025 Gjorgji J. ---

package cmd

import (
	"strings"

	deleteuntaggedimages "ecr-lifecycle-cleaner/internal/deleteUntaggedImages"
	initawsclient "ecr-lifecycle-cleaner/internal/initAwsClient"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Automates the cleanup of untagged images in ECR.",
	Long: `Automates the cleanup of untagged images in Amazon Elastic Container Registry (ECR).

It retrieves all repositories, identifies untagged images that are not referenced by any tagged images,
and deletes those untagged images to help manage storage and maintain a clean registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("[INFO] clean called")

		if repoList != "" {
			repositoryList = strings.Split(repoList, ",")
		}

		ctx := cmd.Context()
		client, account, region, err := initawsclient.NewECRClient(ctx, config.LoadDefaultConfig)
		if err != nil {
			cmd.Printf("[ERROR] Failed to initialize AWS client: %v\n", err)
			return
		}
		cmd.Printf("[INFO] Using AWS account: %s, region: %s\n", account, region)

		var repos []string
		if allRepos {
			repos, err = deleteuntaggedimages.ListRepositories(ctx, client)
			if err != nil {
				cmd.Printf("[ERROR] Failed to list repositories: %v\n", err)
				return
			}
		} else if repoPattern != "" {
			repos, err = deleteuntaggedimages.ListRepositoriesByPattern(ctx, client, repoPattern)
			if err != nil {
				cmd.Printf("[ERROR] Failed to list repositories by pattern: %v\n", err)
				return
			}
		} else {
			repos = repositoryList
		}

		if len(repos) == 0 {
			cmd.Println("[INFO] No repositories to clean.")
			return
		}

		err = deleteuntaggedimages.Main(client, allRepos, repos, repoPattern, dryRun)
		if err != nil {
			cmd.Printf("[ERROR] Failed to clean ECR: %v\n", err)
			return
		}

		cmd.Println("[INFO] Finished ECR untagged images cleanup.")
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
