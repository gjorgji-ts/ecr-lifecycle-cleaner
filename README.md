# ECR Lifecycle Cleaner

A simple command line tool for managing AWS Elastic Container Registry (ECR) repositories.
The purpose of this tool is to automate the process of managing ECR lifecycle policies on a larger scale,
as well as to provide a simple way to clean up old orphaned images from multi-platform builds.

> [!WARNING]
> **Use this tool with caution! There is no undo!**

-----

## Overview

Using `docker buildx` to tag and push multi-arch images to ECR allows for parallel build speeds by building multiple platforms simultaneously. However, this process results in orphaned images that are not deleted by the ECR lifecycle policy.

For example, if you push a multi-arch image with the tag `1.0.0` to ECR, you will get an `Image Index` with the tag `1.0.0` and multiple `Image` artifacts without tag. The `Image Index` is a JSON manifest that points to the `Image` artifacts. ECR lifecycle policies will only delete the tagged `Image Index` and not the `Image` artifacts.

This tool helps you identify and clean up those orphaned images using the `clean` command. It also provides a way to apply lifecycle policies to multiple repositories at once using the `setPolicy` command.

> [!WARNING]
> **This tool will overwrite the existing lifecycle policy with the new one. Make sure to include all the rules in the JSON file.**

-----

## Usage

> [!NOTE]
> Currently, the tool is only tested on Linux and MacOS platforms.
> The Windows platform is supported, but not tested.

### Prerequisites

- AWS CLI installed and configured with the necessary permissions:
  - "sts:GetCallerIdentity" -- Allows the tool to identify the AWS account being used, which is required for the ECR API calls.
  - "ecr:DescribeRepositories" -- Allows the tool to list all the repositories in the account, which is required for the `--allRepos` flag.
  - "ecr:ListImages" -- Allows the tool to list all the images in a repository, which is required for the `clean` command.
  - "ecr:BatchGetImage" -- Allows the tool to get the image details, which is required for the `clean` command.
  - "ecr:BatchDeleteImage" -- Allows the tool to delete the images, which is required for the `clean` command.
  - "ecr:GetLifecyclePolicy" -- Allows the tool to get the existing lifecycle policy, which is required for the `setPolicy` command.
  - "ecr:PutLifecyclePolicy" -- Allows the tool to set the lifecycle policy, which is required for the `setPolicy` command.

### Steps

1. **Download:** Download the latest release for your platform from the [Releases](https://github.com/gjorgji-ts/ecr-lifecycle-cleaner/releases) page.
2. **Verify Checksum (Optional):** Verify the checksum of the downloaded archive. The checksums are available in the `ecr-lifecycle-cleaner_checksums.txt` file.
3. **Unpack and Move Binary:** Unpack the archive and move the binary to a directory in your PATH.
4. **Run Help Command:** Run the binary with the `--help` flag to see the available commands and options.

    ```bash
    ecr-lifecycle-cleaner --help
    ```

5. **Add Shell Completion (Optional):** To add completion for your shell, run the following command:

    ```bash
    ecr-lifecycle-cleaner completion <shell> > /path/to/completion-file
    # e.g. ecr-lifecycle-cleaner completion fish > ~/.config/fish/completions/ecr-lifecycle-cleaner.fish
    ```

### Examples

- **Clean Orphaned Images:**

    ```bash
    ecr-lifecycle-cleaner clean --allRepos
    ```

- **Set Lifecycle Policy:**

    ```bash
    ecr-lifecycle-cleaner setPolicy --policyFile policy.json --allRepos
    ```

- **Dry Run:**

    ```bash
    ecr-lifecycle-cleaner clean --allRepos --dryRun
    # or
    ecr-lifecycle-cleaner setPolicy --policyFile policy.json --allRepos --dryRun
    ```

-----

## Tools Used
- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2)
- [Cobra](https://github.com/spf13/cobra)
- [GoReleaser](https://goreleaser.com)
- [Syft](https://github.com/anchore/syft)

-----

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
