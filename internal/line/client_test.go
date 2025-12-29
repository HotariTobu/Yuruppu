package line_test

import (
	"log/slog"
	"testing"
	"yuruppu/internal/line"

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
		errVariable  string
	}{
		{
			name:         "empty channel token returns ConfigError",
			channelToken: "",
			wantErr:      true,
			errVariable:  "channelToken",
		},
		{
			name:         "whitespace-only channel token returns ConfigError",
			channelToken: "   \t\n  ",
			wantErr:      true,
			errVariable:  "channelToken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := line.NewClient(tt.channelToken, logger)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, c)

				configErr, ok := err.(*line.ConfigError)
				require.True(t, ok, "error should be *line.ConfigError")
				assert.Equal(t, tt.errVariable, configErr.Variable)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, c)
			}
		})
	}
}
