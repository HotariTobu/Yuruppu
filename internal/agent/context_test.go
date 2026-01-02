package agent_test

import (
	"context"
	"testing"
	"yuruppu/internal/agent"

	"github.com/stretchr/testify/assert"
)

func TestWithModelName_And_ModelNameFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		model   string
		wantOK  bool
		wantVal string
	}{
		{
			name:    "set and retrieve model name",
			model:   "gemini-2.0-flash-001",
			wantOK:  true,
			wantVal: "gemini-2.0-flash-001",
		},
		{
			name:    "empty model name is valid",
			model:   "",
			wantOK:  true,
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := agent.WithModelName(context.Background(), tt.model)
			got, ok := agent.ModelNameFromContext(ctx)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantVal, got)
		})
	}
}

func TestModelNameFromContext_NotSet(t *testing.T) {
	t.Parallel()

	got, ok := agent.ModelNameFromContext(context.Background())

	assert.False(t, ok)
	assert.Equal(t, "", got)
}
