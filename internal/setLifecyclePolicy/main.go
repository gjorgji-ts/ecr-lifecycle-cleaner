// Copyright © 2025 Gjorgji J.

package setlifecyclepolicy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// Main is the entry point for setting ECR lifecycle policies.
// It fetches the list of repositories based on the provided parameters and sets the lifecycle policy for each.
func Main(client *ecr.Client, policyText string, allRepos bool, repositoryList []string, repoPattern string) {
	log.SetOutput(os.Stdout)
	log.Println("============================================")
	log.Println("Starting ECR lifecycle policy setup")
	log.Println("============================================")
	ctx := context.TODO()
	if allRepos {
		var err error
		repositoryList, err = getRepositories(ctx, client)
		if err != nil {
			log.Fatalf("[ERROR] Error fetching repositories: %v", err)
		}
	} else if len(repoPattern) > 0 {
		var err error
		repositoryList, err = getRepositoriesByPatterns(ctx, client, repoPattern)
		if err != nil {
			log.Fatalf("[ERROR] Error fetching repositories by patterns: %v", err)
		}
	}
	if len(repositoryList) == 0 {
		log.Println("[INFO] No repositories to set policies for.")
		return
	}
	if err := setPolicyForAll(ctx, client, policyText, repositoryList); err != nil {
		log.Fatalf("[ERROR] Error setting policies: %v", err)
	}
	log.Println("============================================")
	log.Println("Finished ECR lifecycle policy setup")
	log.Println("============================================")
}

func getRepositories(ctx context.Context, client *ecr.Client) ([]string, error) {
	log.Println("[INFO] Fetching list of repositories...")
	var repositories []string
	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get next page of repositories: %w", err)
		}
		for _, repo := range page.Repositories {
			repositories = append(repositories, aws.ToString(repo.RepositoryName))
			log.Printf("[INFO] Found repository: %s", aws.ToString(repo.RepositoryName))
		}
	}
	log.Println("[INFO] Successfully fetched list of repositories.")
	return repositories, nil
}

func getRepositoriesByPatterns(ctx context.Context, client *ecr.Client, repoPattern string) ([]string, error) {
	var repositories []string
	allRepositories, err := getRepositories(ctx, client)
	if err != nil {
		return nil, err
	}

	log.Println("[INFO] Fetching list of repositories by patterns...")
	for _, repo := range allRepositories {
		matched, err := regexp.MatchString(repoPattern, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to match pattern %s: %w", repoPattern, err)
		}
		if matched {
			repositories = append(repositories, repo)
			log.Printf("[INFO] Repository %s matches pattern %s", repo, repoPattern)
		}
	}
	log.Println("[INFO] Successfully fetched list of repositories by patterns.")
	return repositories, nil
}

func getPolicy(ctx context.Context, client *ecr.Client, repository string) error {
	log.Printf("[INFO] Fetching lifecycle policy for repository: %s", repository)
	input := &ecr.GetLifecyclePolicyInput{
		RepositoryName: aws.String(repository),
	}
	resp, err := client.GetLifecyclePolicy(ctx, input)
	if err != nil {
		var notFound *types.LifecyclePolicyNotFoundException
		if errors.As(err, &notFound) {
			log.Printf("[INFO] No lifecycle policy found for repository: %s", repository)
			return nil
		}
		return fmt.Errorf("failed to get lifecycle policy for %s: %w", repository, err)
	}
	log.Printf("[INFO] Lifecycle policy for repository %s: %s", repository, aws.ToString(resp.LifecyclePolicyText))
	return nil
}

func setPolicy(ctx context.Context, client *ecr.Client, repository string, policyText string) error {
	log.Printf("[INFO] Setting lifecycle policy for repository: %s", repository)
	input := &ecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(repository),
		LifecyclePolicyText: aws.String(policyText),
	}
	resp, err := client.PutLifecyclePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set lifecycle policy for %s: %w", repository, err)
	}
	log.Printf("[INFO] Successfully set lifecycle policy for repository %s: %s", repository, aws.ToString(resp.LifecyclePolicyText))
	return nil
}

func setPolicyForAll(ctx context.Context, client *ecr.Client, policyText string, repoList []string) error {
	log.Println("[INFO] Starting to set lifecycle policies for specified repositories...")

	var wg sync.WaitGroup
	for _, repository := range repoList {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			if err := setPolicy(ctx, client, repo, policyText); err != nil {
				log.Printf("[ERROR] Error setting policy for repository %s: %v", repo, err)
			}
			if err := getPolicy(ctx, client, repo); err != nil {
				log.Printf("[ERROR] Error getting policy for repository %s: %v", repo, err)
			}
		}(repository)
	}
	wg.Wait()
	log.Println("[INFO] Finished setting lifecycle policies for specified repositories.")
	return nil
}
