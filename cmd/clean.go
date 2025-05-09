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

		for _, repo := range repos {
			orphans, taggedCount, orphanCount, err := deleteuntaggedimages.ImagesToDelete(ctx, repo, client)
			if err != nil {
				cmd.Printf("[ERROR] %s: %v\n", repo, err)
				continue
			}
			cmd.Printf("[INFO] Repository: %s - Found %d tagged and %d untagged images. Deleting %d orphans.\n", repo, taggedCount, orphanCount, len(orphans))
			if len(orphans) > 0 {
				deleted, failed, err := deleteuntaggedimages.DeleteImages(ctx, repo, orphans, client, dryRun)
				if err != nil {
					cmd.Printf("[ERROR] Failed to delete images in %s: %v\n", repo, err)
					continue
				}
				if dryRun {
					cmd.Printf("[DRY RUN] Would have deleted %d images in %s\n", len(orphans), repo)
				} else {
					cmd.Printf("[INFO] Deleted %d images, failed to delete %d images in %s\n", deleted, failed, repo)
				}
			} else {
				cmd.Printf("[INFO] Repository: %s - Nothing to delete\n", repo)
			}
		}
		cmd.Println("[INFO] Finished ECR untagged images cleanup.")
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
