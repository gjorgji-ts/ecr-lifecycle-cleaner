// Copyright © 2025 Gjorgji J.

package deleteuntaggedimages

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// Main is the entry point for deleting untagged images from ECR repositories.
// It fetches the list of repositories and deletes the untagged images from each.
func Main(client *ecr.Client, allRepos bool, repositoryList []string, repoPattern string) {
	log.SetOutput(os.Stdout)
	log.Println("============================================")
	log.Println("Starting ECR untagged images cleanup")
	log.Println("============================================")

	ctx := context.TODO()
	if allRepos {
		var err error
		repositoryList, err = getRepositories(ctx, client)
		if err != nil {
			log.Fatalf("Error fetching repositories: %v", err)
		}
	} else if len(repoPattern) > 0 {
		var err error
		repositoryList, err = getRepositoriesByPatterns(ctx, client, repoPattern)
		if err != nil {
			log.Fatalf("Error fetching repositories by patterns: %v", err)
		}
	}

	if len(repositoryList) == 0 {
		log.Println("[INFO] No repositories to clean.")
		return
	}

	if err := cleanECR(ctx, client, repositoryList); err != nil {
		log.Fatalf("Error cleaning ECR: %v", err)
	}
	log.Println("============================================")
	log.Println("Finished ECR untagged images cleanup")
	log.Println("============================================")
}

func getRepositories(ctx context.Context, client *ecr.Client) ([]string, error) {
	var repositories []string
	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe repositories: %w", err)
		}
		for _, repo := range page.Repositories {
			repositories = append(repositories, aws.ToString(repo.RepositoryName))
		}
	}
	return repositories, nil
}

func getRepositoriesByPatterns(ctx context.Context, client *ecr.Client, repoPattern string) ([]string, error) {
	var repositories []string
	allRepositories, err := getRepositories(ctx, client)
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

func getImages(ctx context.Context, repository string, client *ecr.Client) (map[string][]string, error) {
	images := map[string][]string{"tagged": {}, "orphan": {}}
	paginator := ecr.NewListImagesPaginator(client, &ecr.ListImagesInput{RepositoryName: aws.String(repository)})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list images for repository %s: %w", repository, err)
		}
		for _, image := range page.ImageIds {
			if image.ImageTag != nil {
				images["tagged"] = append(images["tagged"], aws.ToString(image.ImageDigest))
			} else {
				images["orphan"] = append(images["orphan"], aws.ToString(image.ImageDigest))
			}
		}
	}
	return images, nil
}

func getChildImages(ctx context.Context, repository string, images []string, client *ecr.Client) ([]string, error) {
	var children []string
	imageIds := []types.ImageIdentifier{}
	for _, digest := range images {
		imageIds = append(imageIds, types.ImageIdentifier{ImageDigest: aws.String(digest)})
	}
	input := &ecr.BatchGetImageInput{
		RepositoryName: aws.String(repository),
		ImageIds:       imageIds,
	}
	result, err := client.BatchGetImage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get images for repository %s: %w", repository, err)
	}
	for _, image := range result.Images {
		var manifest map[string]interface{}
		err := json.Unmarshal([]byte(aws.ToString(image.ImageManifest)), &manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal image manifest for repository %s: %w", repository, err)
		}
		if manifests, ok := manifest["manifests"].([]interface{}); ok {
			for _, m := range manifests {
				if digest, ok := m.(map[string]interface{})["digest"].(string); ok {
					children = append(children, digest)
				}
			}
		}
	}
	return children, nil
}

func partitionList(lst []string, size int) [][]string {
	var partitions [][]string
	for i := 0; i < len(lst); i += size {
		end := i + size
		if end > len(lst) {
			end = len(lst)
		}
		partitions = append(partitions, lst[i:end])
	}
	return partitions
}

func getImagesToDelete(ctx context.Context, repository string, client *ecr.Client, logMessages *[]string, mu *sync.Mutex) ([]string, error) {
	images, err := getImages(ctx, repository, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get images for repository %s: %w", repository, err)
	}
	logMessage := fmt.Sprintf("[INFO] Repository: %s - Found %d tagged and %d untagged images", repository, len(images["tagged"]), len(images["orphan"]))
	mu.Lock()
	*logMessages = append(*logMessages, logMessage)
	mu.Unlock()

	for _, part := range partitionList(images["tagged"], 100) {
		logMessage = fmt.Sprintf("[INFO] Repository: %s - Finding children of the tagged images", repository)
		mu.Lock()
		*logMessages = append(*logMessages, logMessage)
		mu.Unlock()

		children, err := getChildImages(ctx, repository, part, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get child images for repository %s: %w", repository, err)
		}
		orphanImages := []string{}
		for _, orphan := range images["orphan"] {
			found := false
			for _, child := range children {
				if orphan == child {
					found = true
					break
				}
			}
			if !found {
				orphanImages = append(orphanImages, orphan)
			}
		}
		images["orphan"] = orphanImages
	}
	return images["orphan"], nil
}

func deleteImages(ctx context.Context, repository string, images []string, client *ecr.Client) error {
	deleted := 0
	failed := 0
	for _, part := range partitionList(images, 100) {
		imageIds := []types.ImageIdentifier{}
		for _, digest := range part {
			imageIds = append(imageIds, types.ImageIdentifier{ImageDigest: aws.String(digest)})
		}
		input := &ecr.BatchDeleteImageInput{
			RepositoryName: aws.String(repository),
			ImageIds:       imageIds,
		}
		result, err := client.BatchDeleteImage(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to batch delete images for repository %s: %w", repository, err)
		}
		for _, failure := range result.Failures {
			log.Printf("[ERROR] Repository: %s - Failed to delete %s: %s - %s", repository, aws.ToString(failure.ImageId.ImageDigest), string(failure.FailureCode), aws.ToString(failure.FailureReason))
		}
		deleted += len(result.ImageIds)
		failed += len(result.Failures)
	}
	log.Printf("[INFO] Repository: %s - Deleted %d images, failed to delete %d images", repository, deleted, failed)
	return nil
}

func cleanECR(ctx context.Context, client *ecr.Client, repositories []string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	var logMessages []string

	for _, repository := range repositories {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			logMessage := fmt.Sprintf("[INFO] Checking repository: %s", repo)
			mu.Lock()
			logMessages = append(logMessages, logMessage)
			mu.Unlock()

			images, err := getImagesToDelete(ctx, repo, client, &logMessages, &mu)
			if err != nil {
				logMessage = fmt.Sprintf("[ERROR] Repository: %s - Failed to get images to delete: %v", repo, err)
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			if len(images) > 0 {
				logMessage = fmt.Sprintf("[INFO] Repository: %s - Deleting %d images", repo, len(images))
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				mu.Unlock()

				err := deleteImages(ctx, repo, images, client)
				if err != nil {
					logMessage = fmt.Sprintf("[ERROR] Repository: %s - Failed to delete images: %v", repo, err)
					mu.Lock()
					logMessages = append(logMessages, logMessage)
					errs = append(errs, err)
					mu.Unlock()
				}
			} else {
				logMessage = fmt.Sprintf("[INFO] Repository: %s - Nothing to delete", repo)
				mu.Lock()
				logMessages = append(logMessages, logMessage)
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
		return fmt.Errorf("encountered errors during cleanup: %v", errs)
	}
	return nil
}
