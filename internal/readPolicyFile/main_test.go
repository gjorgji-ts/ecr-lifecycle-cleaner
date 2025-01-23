// Copyright © 2025 Gjorgji J.

package readpolicyfile

import (
	"os"
	"testing"
)

func TestReadPolicyFile(t *testing.T) {
	// Test case: Valid policy file
	t.Run("Valid policy file", func(t *testing.T) {
		// Create a temporary file with valid JSON content
		tmpFile, err := os.CreateTemp("", "policy-*.json")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		policyContent := `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}`
		if _, err := tmpFile.Write([]byte(policyContent)); err != nil {
			t.Fatalf("Failed to write to temporary file: %v", err)
		}
		tmpFile.Close()

		// Call the function and check the result
		result, err := ReadPolicyFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != policyContent {
			t.Fatalf("Expected %s, got %s", policyContent, result)
		}
	})

	// Test case: Non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		_, err := ReadPolicyFile("non-existent-file.json")
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		expectedError := "failed to open policy file"
		if err.Error()[:len(expectedError)] != expectedError {
			t.Fatalf("Expected error to start with %s, got %v", expectedError, err)
		}
	})

	// Test case: Invalid JSON
	t.Run("Invalid JSON", func(t *testing.T) {
		// Create a temporary file with invalid JSON content
		tmpFile, err := os.CreateTemp("", "policy-*.json")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		invalidJSONContent := `{"Version": "2012-10-17", "Statement": [`
		if _, err := tmpFile.Write([]byte(invalidJSONContent)); err != nil {
			t.Fatalf("Failed to write to temporary file: %v", err)
		}
		tmpFile.Close()

		// Call the function and check the result
		_, err = ReadPolicyFile(tmpFile.Name())
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		expectedError := "invalid JSON in policy file"
		if err.Error()[:len(expectedError)] != expectedError {
			t.Fatalf("Expected error to start with %s, got %v", expectedError, err)
		}
	})
}
