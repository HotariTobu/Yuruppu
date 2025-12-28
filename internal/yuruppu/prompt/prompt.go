// Package prompt provides the system prompt for Yuruppu.
package prompt

import _ "embed"

//go:embed system.txt
var SystemPrompt string
