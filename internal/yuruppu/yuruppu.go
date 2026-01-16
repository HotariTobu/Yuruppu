// Package yuruppu provides the Yuruppu character system prompt.
package yuruppu

import (
	_ "embed"
	"yuruppu/internal/bot"
)

//go:embed character.txt
var CharacterPrompt string

// GetSystemPrompt returns the system prompt with the character prompt injected.
func GetSystemPrompt() (string, error) {
	return bot.BuildSystemPrompt(CharacterPrompt)
}
