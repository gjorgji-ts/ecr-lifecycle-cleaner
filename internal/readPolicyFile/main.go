// --- Copyright © 2025 Gjorgji J. ---

package readpolicyfile

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// --- reads the content of a policy file and returns it as a string ---
func ReadPolicyFile(filePath string) (string, error) {
	log.Println("============================================")
	log.Printf("[INFO] Opening policy file: %s", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] Failed to open policy file: %s, error: %v", filePath, err)
		return "", fmt.Errorf("failed to open policy file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("[WARN] Failed to close policy file: %v", closeErr)
		}
	}()

	log.Printf("[INFO] Reading policy file: %s", filePath)
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[ERROR] Failed to read policy file: %s, error: %v", filePath, err)
		return "", fmt.Errorf("failed to read policy file: %w", err)
	}

	log.Printf("[INFO] Validating JSON content of policy file: %s", filePath)
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonObj); err != nil {
		log.Printf("[ERROR] Invalid JSON in policy file: %s, error: %v", filePath, err)
		return "", fmt.Errorf("invalid JSON in policy file: %w", err)
	}

	log.Printf("[INFO] Successfully read and validated policy file: %s", filePath)
	log.Println("============================================")
	return string(bytes), nil
}

// --- reads and validates policy file, no logging or side effects ---
func ReadPolicyFilePure(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open policy file: %w", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read policy file: %w", err)
	}

	var jsonObj map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonObj); err != nil {
		return "", fmt.Errorf("invalid JSON in policy file: %w", err)
	}

	return string(bytes), nil
}
