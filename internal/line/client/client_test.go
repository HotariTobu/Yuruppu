package client_test

import (
	"log/slog"
	"testing"
	"yuruppu/internal/line/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewClient Tests
// =============================================================================

func TestNewClient(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)

	tests := []struct {
		name         string
		channelToken string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "empty channel token returns error",
			channelToken: "",
			wantErr:      true,
			errContains:  "channelToken",
		},
		{
			name:         "whitespace-only channel token returns error",
			channelToken: "   \t\n  ",
			wantErr:      true,
			errContains:  "channelToken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := client.NewClient(tt.channelToken, logger)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, c)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, c)
			}
		})
	}
}
