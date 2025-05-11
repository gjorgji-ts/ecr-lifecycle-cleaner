// --- Copyright © 2025 Gjorgji J. ---

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Errorf("Expected help output, got: %s", out)
	}
}

func TestCleanCmd_AllReposFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"clean", "--allRepos", "--dryRun"})
	allRepos = true
	repositoryList = nil
	repoPattern = ""
	dryRun = true
	_ = rootCmd.Execute()
	out := buf.String()
	if !strings.Contains(out, "clean called") {
		t.Errorf("Expected clean called output, got: %s", out)
	}
}

func TestSetPolicyCmd_MissingRequiredFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"setPolicy"})
	// --- The command should fail due to missing required flag policyFile ---
	err := rootCmd.Execute()
	if err == nil {
		t.Errorf("Expected error for missing required flag, got nil")
	}
	out := buf.String()
	if !strings.Contains(out, "required flag") && !strings.Contains(out, "policyFile") {
		t.Errorf("Expected required flag error, got: %s", out)
	}
}
