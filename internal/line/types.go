package line

// ConfigError represents an error related to missing or invalid configuration.
type ConfigError struct {
	Variable string
}

func (e *ConfigError) Error() string {
	return "Missing required configuration: " + e.Variable
}
