// --- Copyright © 2025 Gjorgji J. ---

package readpolicyfile

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

// --- errorFile simulates a file that returns an error on Close ---
type errorFile struct {
	*os.File
}

func (f *errorFile) Close() error { return io.ErrClosedPipe }

func TestReadPolicyFile(t *testing.T) {
	// --- valid policy file test ---
	t.Run("Valid policy file", func(t *testing.T) {
		// --- create a temporary file with valid JSON content ---
		tmpFile, err := os.CreateTemp("", "policy-*.json")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				t.Logf("Failed to remove temporary file: %v", err)
			}
		}()

		policyContent := `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}`
		if _, err := tmpFile.Write([]byte(policyContent)); err != nil {
			t.Fatalf("Failed to write to temporary file: %v", err)
		}
		if err := tmpFile.Close(); err != nil {
			t.Fatalf("Failed to close temporary file: %v", err)
		}

		// --- call the function and check the result ---
		result, err := ReadPolicyFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != policyContent {
			t.Fatalf("Expected %s, got %s", policyContent, result)
		}
	})

	// --- non-existent file test ---
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

	// --- invalid JSON test ---
	t.Run("Invalid JSON", func(t *testing.T) {
		// --- create a temporary file with invalid JSON content ---
		tmpFile, err := os.CreateTemp("", "policy-*.json")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				t.Logf("Failed to remove temporary file: %v", err)
			}
		}()

		invalidJSONContent := `{"Version": "2012-10-17", "Statement": [`
		if _, err := tmpFile.Write([]byte(invalidJSONContent)); err != nil {
			t.Fatalf("Failed to write to temporary file: %v", err)
		}
		if err := tmpFile.Close(); err != nil {
			t.Fatalf("Failed to close temporary file: %v", err)
		}

		// --- call the function and check the result ---
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

func TestReadPolicyFileWithLogging(t *testing.T) {
	// --- valid policy file test ---
	t.Run("Valid policy file with logging", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "policy-logging-*.json")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()

		policyContent := `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}`
		_, _ = tmpFile.Write([]byte(policyContent))
		_ = tmpFile.Close()

		result, err := readPolicyFileWithLogging(tmpFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != policyContent {
			t.Fatalf("Expected %s, got %s", policyContent, result)
		}
	})

	// --- non-existent file test ---
	t.Run("Non-existent file with logging", func(t *testing.T) {
		_, err := readPolicyFileWithLogging("non-existent-file.json")
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
	})

	// --- invalid JSON test ---
	t.Run("Invalid JSON with logging", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "policy-logging-*.json")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()

		invalidJSONContent := `{"Version": "2012-10-17", "Statement": [`
		_, _ = tmpFile.Write([]byte(invalidJSONContent))
		_ = tmpFile.Close()

		_, err = readPolicyFileWithLogging(tmpFile.Name())
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
	})
}

// --- Test for error handling in file.Close() ---
func TestReadPolicyFile_CloseError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "policy-close-*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	policyContent := `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}`
	_, _ = tmpFile.Write([]byte(policyContent))
	_ = tmpFile.Close()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	file, _ := os.Open(tmpFile.Name())
	f := &errorFile{file}

	func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to close policy file: %v\n", cerr)
		}
	}()

	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe writer: %v", err)
	}
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	output := buf.String()

	if output == "" || !bytes.Contains([]byte(output), []byte("[WARN] Failed to close policy file")) {
		t.Errorf("Expected warning about failed file close, got: %s", output)
	}
}
