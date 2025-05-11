// --- Copyright © 2025 Gjorgji J. ---

package cmd

import (
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
		cmd.Println("[INFO] setPolicy called")

		if repoList != "" {
			repositoryList = strings.Split(repoList, ",")
		}

		ctx := cmd.Context()
		policyText, err := readpolicyfile.ReadPolicyFile(policyFile)
		if err != nil {
			cmd.Printf("[ERROR] Reading policy file: %v\n", err)
			return
		}

		client, account, region, err := initawsclient.NewECRClient(ctx, config.LoadDefaultConfig)
		if err != nil {
			cmd.Printf("[ERROR] Failed to initialize AWS client: %v\n", err)
			return
		}
		cmd.Printf("[INFO] Using AWS account: %s, region: %s\n", account, region)

		var repos []string
		if allRepos {
			repos, err = setlifecyclepolicy.GetRepositories(ctx, client)
			if err != nil {
				cmd.Printf("[ERROR] Failed to list repositories: %v\n", err)
				return
			}
		} else if repoPattern != "" {
			repos, err = setlifecyclepolicy.GetRepositoriesByPattern(ctx, client, repoPattern)
			if err != nil {
				cmd.Printf("[ERROR] Failed to list repositories by pattern: %v\n", err)
				return
			}
		} else {
			repos = repositoryList
		}

		if len(repos) == 0 {
			cmd.Println("[INFO] No repositories to set policies for.")
			return
		}

		err = setlifecyclepolicy.Main(client, policyText, allRepos, repos, repoPattern, dryRun)
		if err != nil {
			cmd.Printf("[ERROR] Failed to set lifecycle policies: %v\n", err)
			return
		}

		cmd.Println("[INFO] Finished ECR lifecycle policy setup.")
	},
}

func init() {
	rootCmd.AddCommand(setPolicyCmd)

	setPolicyCmd.Flags().StringVarP(&policyFile, "policyFile", "f", "", "path to the JSON file containing the lifecycle policy")
	setPolicyCmd.MarkFlagRequired("policyFile") // nolint:errcheck
}
