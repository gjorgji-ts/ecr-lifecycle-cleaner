// --- Copyright © 2025 Gjorgji J. ---

package deleteuntaggedimages

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// --- ECRAPI defines the subset of ecr.Client methods used for testability ---
type ECRAPI interface {
	DescribeRepositories(ctx context.Context, in *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	ListImages(ctx context.Context, in *ecr.ListImagesInput, optFns ...func(*ecr.Options)) (*ecr.ListImagesOutput, error)
	BatchGetImage(ctx context.Context, in *ecr.BatchGetImageInput, optFns ...func(*ecr.Options)) (*ecr.BatchGetImageOutput, error)
	BatchDeleteImage(ctx context.Context, in *ecr.BatchDeleteImageInput, optFns ...func(*ecr.Options)) (*ecr.BatchDeleteImageOutput, error)
}

// --- the entry point for deleting untagged images from ECR repositories ---
// --- it fetches the list of repositories and deletes the untagged images from each ---
func Main(client *ecr.Client, allRepos bool, repositoryList []string, repoPattern string, dryRun bool) error {
	ctx := context.TODO()
	if allRepos {
		var err error
		repositoryList, err = getRepositories(ctx, client)
		if err != nil {
			return err
		}
	} else if len(repoPattern) > 0 {
		var err error
		repositoryList, err = getRepositoriesByPatterns(ctx, client, repoPattern)
		if err != nil {
			return err
		}
	}

	if len(repositoryList) == 0 {
		return nil
	}

	if err := CleanECRWithLogging(ctx, client, repositoryList, dryRun); err != nil {
		return err
	}
	return nil
}

// --- returns all repository names ---
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

// --- returns repositories matching a pattern ---
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

// --- returns a map of tagged and orphan image digests ---
func getImages(ctx context.Context, repository string, client ECRAPI) (map[string][]string, error) {
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

// --- returns child image digests for a set of images ---
func getChildImages(ctx context.Context, repository string, images []string, client ECRAPI) ([]string, error) {
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

// --- splits a list into chunks of a given size ---
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

// --- filter orphans not referenced by children ---
func filterOrphans(orphans, children []string) []string {
	result := make([]string, 0, len(orphans))
	childSet := make(map[string]struct{}, len(children))
	for _, c := range children {
		childSet[c] = struct{}{}
	}
	for _, orphan := range orphans {
		if _, found := childSet[orphan]; !found {
			result = append(result, orphan)
		}
	}
	return result
}

// --- returns orphan images to delete ---
func imagesToDeleteWithLogging(ctx context.Context, repository string, client ECRAPI, logMessages *[]string, mu *sync.Mutex) ([]string, error) {
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
		images["orphan"] = filterOrphans(images["orphan"], children)
	}
	return images["orphan"], nil
}

// --- deletes images from a repository ---
func deleteImagesWithLogging(ctx context.Context, repository string, images []string, client ECRAPI, dryRun bool, logMessages *[]string, mu *sync.Mutex) error {
	if dryRun {
		logMessage := fmt.Sprintf("[DRY RUN] Would delete %d images from repository: %s", len(images), repository)
		mu.Lock()
		*logMessages = append(*logMessages, logMessage)
		mu.Unlock()
		return nil
	}
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
		logMessage := fmt.Sprintf("[INFO] Repository: %s - Deleting %d images", repository, len(part))
		mu.Lock()
		*logMessages = append(*logMessages, logMessage)
		mu.Unlock()
		result, err := client.BatchDeleteImage(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to batch delete images for repository %s: %w", repository, err)
		}
		for _, failure := range result.Failures {
			logMessage = fmt.Sprintf("[ERROR] Repository: %s - Failed to delete %s: %s - %s", repository, aws.ToString(failure.ImageId.ImageDigest), string(failure.FailureCode), aws.ToString(failure.FailureReason))
			mu.Lock()
			*logMessages = append(*logMessages, logMessage)
			mu.Unlock()
		}
		deleted += len(result.ImageIds)
		failed += len(result.Failures)
	}
	logMessage := fmt.Sprintf("[INFO] Repository: %s - Deleted %d images, failed to delete %d images", repository, deleted, failed)
	mu.Lock()
	*logMessages = append(*logMessages, logMessage)
	mu.Unlock()
	return nil
}

// --- runs the cleanup process for all repositories ---
func CleanECRWithLogging(ctx context.Context, client ECRAPI, repositories []string, dryRun bool) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	var logMessages []string

	for _, repository := range repositories {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			if dryRun {
				logMessage := fmt.Sprintf("[DRY RUN] Would delete untagged images from repository: %s", repo)
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				mu.Unlock()
				return
			}
			logMessage := fmt.Sprintf("[INFO] Checking repository: %s", repo)
			mu.Lock()
			logMessages = append(logMessages, logMessage)
			mu.Unlock()

			images, err := imagesToDeleteWithLogging(ctx, repo, client, &logMessages, &mu)
			if err != nil {
				logMessage = fmt.Sprintf("[ERROR] Repository: %s - Failed to get images to delete: %v", repo, err)
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			if len(images) > 0 {
				mu.Lock()
				logMessages = append(logMessages, logMessage)
				mu.Unlock()

				err := deleteImagesWithLogging(ctx, repo, images, client, dryRun, &logMessages, &mu)
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

// --- returns repositories, error only ---
func ListRepositories(ctx context.Context, client ECRAPI) ([]string, error) {
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

// --- returns repositories matching pattern ---
func ListRepositoriesByPattern(ctx context.Context, client ECRAPI, repoPattern string) ([]string, error) {
	allRepositories, err := ListRepositories(ctx, client)
	if err != nil {
		return nil, err
	}
	var repositories []string
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

// --- returns map of tagged/orphan digests ---
func listImages(ctx context.Context, repository string, client ECRAPI) (map[string][]string, error) {
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

// --- returns child digests ---
func listChildImages(ctx context.Context, repository string, images []string, client ECRAPI) ([]string, error) {
	var children []string
	imageIds := make([]types.ImageIdentifier, len(images))
	for i, digest := range images {
		imageIds[i] = types.ImageIdentifier{ImageDigest: aws.String(digest)}
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

// --- returns orphan digests to delete ---
func imagesToDelete(ctx context.Context, repository string, client ECRAPI) ([]string, int, int, error) {
	images, err := listImages(ctx, repository, client)
	if err != nil {
		return nil, 0, 0, err
	}
	tagged := images["tagged"]
	orphans := images["orphan"]
	for _, part := range partitionList(tagged, 100) {
		children, err := listChildImages(ctx, repository, part, client)
		if err != nil {
			return nil, 0, 0, err
		}
		orphans = filterOrphans(orphans, children)
	}
	return orphans, len(tagged), len(images["orphan"]), nil
}

// --- returns (deleted, failed, error) ---
func deleteImages(ctx context.Context, repository string, images []string, client ECRAPI, dryRun bool) (int, int, error) {
	deleted := 0
	failed := 0
	for _, part := range partitionList(images, 100) {
		imageIds := make([]types.ImageIdentifier, len(part))
		for i, digest := range part {
			imageIds[i] = types.ImageIdentifier{ImageDigest: aws.String(digest)}
		}
		if dryRun {
			continue
		}
		input := &ecr.BatchDeleteImageInput{
			RepositoryName: aws.String(repository),
			ImageIds:       imageIds,
		}
		result, err := client.BatchDeleteImage(ctx, input)
		if err != nil {
			return deleted, failed, fmt.Errorf("failed to batch delete images for repository %s: %w", repository, err)
		}
		deleted += len(result.ImageIds)
		failed += len(result.Failures)
	}
	return deleted, failed, nil
}
