// --- Copyright © 2025 Gjorgji J. ---

package cmd

import (
	"fmt"
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

		ctx := cmd.Context()
		policyText, err := readpolicyfile.ReadPolicyFilePure(policyFile)
		if err != nil {
			fmt.Printf("[ERROR] Reading policy file: %v\n", err)
			return
		}

		client, account, region, err := initawsclient.NewECRClient(ctx, config.LoadDefaultConfig)
		if err != nil {
			fmt.Printf("[ERROR] Failed to initialize AWS client: %v\n", err)
			return
		}
		fmt.Printf("[INFO] Using AWS account: %s, region: %s\n", account, region)

		var repos []string
		if allRepos {
			repos, err = setlifecyclepolicy.GetRepositories(ctx, client)
			if err != nil {
				fmt.Printf("[ERROR] Failed to list repositories: %v\n", err)
				return
			}
		} else if repoPattern != "" {
			repos, err = setlifecyclepolicy.GetRepositoriesByPattern(ctx, client, repoPattern)
			if err != nil {
				fmt.Printf("[ERROR] Failed to list repositories by pattern: %v\n", err)
				return
			}
		} else {
			repos = repositoryList
		}

		if len(repos) == 0 {
			fmt.Println("[INFO] No repositories to set policies for.")
			return
		}

		results := setlifecyclepolicy.SetPolicyForAll(ctx, client, policyText, repos, dryRun)
		for repo, err := range results {
			if err != nil {
				fmt.Printf("[ERROR] Failed to set policy for %s: %v\n", repo, err)
			} else {
				if dryRun {
					fmt.Printf("[DRY RUN] Would have set policy for %s\n", repo)
				} else {
					fmt.Printf("[INFO] Successfully set policy for %s\n", repo)
				}
			}
		}
		fmt.Println("[INFO] Finished ECR lifecycle policy setup.")
	},
}

func init() {
	rootCmd.AddCommand(setPolicyCmd)

	setPolicyCmd.Flags().StringVarP(&policyFile, "policyFile", "f", "", "path to the JSON file containing the lifecycle policy")
	setPolicyCmd.MarkFlagRequired("policyFile") // nolint:errcheck
}
