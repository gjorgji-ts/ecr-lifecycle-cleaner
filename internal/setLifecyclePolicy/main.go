// Copyright © 2025 Gjorgji J.

package setlifecyclepolicy

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
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

	sort.Strings(repositoryList)

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
		}
	}
	sort.Strings(repositories)
	for _, repo := range repositories {
		log.Printf("[INFO] Found repository: %s", repo)
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
		}
	}
	sort.Strings(repositories)
	for _, repo := range repositories {
		log.Printf("[INFO] Repository %s matches pattern %s", repo, repoPattern)
	}
	log.Println("[INFO] Successfully fetched list of repositories by patterns.")
	return repositories, nil
}

func setPolicy(ctx context.Context, client *ecr.Client, repository string, policyText string) (string, error) {
	logMsg := fmt.Sprintf("[INFO] Setting lifecycle policy for repository: %s", repository)
	input := &ecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(repository),
		LifecyclePolicyText: aws.String(policyText),
	}
	resp, err := client.PutLifecyclePolicy(ctx, input)
	if err != nil {
		return logMsg, fmt.Errorf("failed to set lifecycle policy for %s: %w", repository, err)
	}
	logMsg += fmt.Sprintf("\n[INFO] Successfully set lifecycle policy for repository %s:\n %s", repository, aws.ToString(resp.LifecyclePolicyText))
	return logMsg, nil
}

func setPolicyForAll(ctx context.Context, client *ecr.Client, policyText string, repoList []string) error {
	log.Println("[INFO] Starting to set lifecycle policies for specified repositories...")

	var wg sync.WaitGroup
	logMap := sync.Map{}

	for _, repository := range repoList {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			logs := []string{}
			if logMsg, err := setPolicy(ctx, client, repo, policyText); err != nil {
				logs = append(logs, fmt.Sprintf("[ERROR] Error setting policy for repository %s: %v", repo, err))
			} else {
				logs = append(logs, logMsg)
			}
			logMap.Store(repo, strings.Join(logs, "\n"))
		}(repository)
	}

	wg.Wait()

	for _, repo := range repoList {
		if logMsg, ok := logMap.Load(repo); ok {
			log.Println(logMsg)
		}
	}

	log.Println("[INFO] Finished setting lifecycle policies for specified repositories.")
	return nil
}
