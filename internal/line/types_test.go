package line_test

import (
	"testing"

	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// ConfigError Tests
// =============================================================================

func TestConfigError_Message(t *testing.T) {
	t.Parallel()

	err := &line.ConfigError{Variable: "channelToken"}
	expected := "Missing required configuration: channelToken"
	assert.Equal(t, expected, err.Error())
}
