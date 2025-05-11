// --- Copyright © 2025 Gjorgji J. ---

package setlifecyclepolicy

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

// --- the entry point for setting ECR lifecycle policies ---
// --- It fetches the list of repositories based on the provided parameters and sets the lifecycle policy for each ---
func Main(client *ecr.Client, policyText string, allRepos bool, repositoryList []string, repoPattern string, dryRun bool) error {
	ctx := context.TODO()
	if allRepos {
		var err error
		repositoryList, err = GetRepositories(ctx, client)
		if err != nil {
			return err
		}
	} else if len(repoPattern) > 0 {
		var err error
		repositoryList, err = GetRepositoriesByPattern(ctx, client, repoPattern)
		if err != nil {
			return err
		}
	}
	if len(repositoryList) == 0 {
		return nil
	}

	if err := setPolicyForAll(ctx, client, policyText, repositoryList, dryRun); err != nil {
		return err
	}
	return nil
}

// --- returns all repository names ---
func GetRepositories(ctx context.Context, client *ecr.Client) ([]string, error) {
	var repositories []string
	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get next page of repositories: %w", err)
		}
		for _, repo := range page.Repositories {
			repositories = append(repositories, aws.ToString(repo.RepositoryName))
		}
	}
	return repositories, nil
}

// --- returns repositories matching a pattern ---
func GetRepositoriesByPattern(ctx context.Context, client *ecr.Client, repoPattern string) ([]string, error) {
	var repositories []string
	allRepositories, err := GetRepositories(ctx, client)
	if err != nil {
		return nil, err
	}

	for _, repo := range allRepositories {
		matched, err := regexp.MatchString(repoPattern, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to match pattern %s: %w", repoPattern, err)
		}
		if matched {
			repositories = append(repositories, repo)
		}
	}
	return repositories, nil
}

// --- sets the lifecycle policy for a repository ---
func setPolicy(ctx context.Context, client *ecr.Client, repository string, policyText string, dryRun bool) (string, error) {
	if dryRun {
		return fmt.Sprintf("[DRY RUN] Would set lifecycle policy for repository: %s", repository), nil
	}
	input := &ecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(repository),
		LifecyclePolicyText: aws.String(policyText),
	}
	resp, err := client.PutLifecyclePolicy(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to set lifecycle policy for %s: %w", repository, err)
	}
	return fmt.Sprintf("[INFO] Successfully set lifecycle policy for repository %s:\n %s", repository, aws.ToString(resp.LifecyclePolicyText)), nil
}

// --- sets the policy for all repositories in the list ---
func setPolicyForAll(ctx context.Context, client *ecr.Client, policyText string, repoList []string, dryRun bool) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	var logMessages []string

	for _, repository := range repoList {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			if dryRun {
				logMessage := fmt.Sprintf("[DRY RUN] Would set lifecycle policy for repository: %s", repo)
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				mu.Unlock()
				return
			}
			logMessage := fmt.Sprintf("[INFO] Setting policy for repository: %s", repo)
			mu.Lock()
			logMessages = append(logMessages, logMessage)
			mu.Unlock()

			if logMsg, err := setPolicy(ctx, client, repo, policyText, dryRun); err != nil {
				logMessage = fmt.Sprintf("[ERROR] Repository: %s - Failed to set policy: %v", repo, err)
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				errs = append(errs, err)
				mu.Unlock()
			} else {
				mu.Lock()
				logMessages = append(logMessages, logMsg)
				mu.Unlock()
			}
		}(repository)
	}
	wg.Wait()

	sort.Slice(logMessages, func(i, j int) bool {
		return logMessages[i] < logMessages[j]
	})

	for _, logMessage := range logMessages {
		log.Println(logMessage)
	}

	if len(errs) > 0 {
		return fmt.Errorf("encountered errors during policy setup: %v", errs)
	}
	return nil
}
