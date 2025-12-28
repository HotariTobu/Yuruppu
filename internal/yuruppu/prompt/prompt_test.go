package prompt

import "testing"

// TestSystemPrompt_NotEmpty verifies SystemPrompt is embedded and non-empty.
// This catches issues where system.txt is missing or empty at build time.
func TestSystemPrompt_NotEmpty(t *testing.T) {
	if SystemPrompt == "" {
		t.Fatal("SystemPrompt should not be empty - check that system.txt exists and contains content")
	}
}
