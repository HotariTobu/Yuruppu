# Input variables for Yuruppu infrastructure

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for resources"
  type        = string
  default     = "asia-northeast1"
}

variable "github_owner" {
  description = "GitHub repository owner"
  type        = string
}

variable "github_repo" {
  description = "GitHub repository name"
  type        = string
  default     = "Yuruppu"
}

variable "github_connection" {
  description = "Cloud Build connection name for GitHub"
  type        = string
}

variable "github_app_installation_id" {
  description = "GitHub App installation ID for Cloud Build connection"
  type        = number
}

variable "log_level" {
  description = "Log level (DEBUG, INFO, WARN, ERROR)"
  type        = string
  default     = "INFO"

  validation {
    condition     = contains(["DEBUG", "INFO", "WARN", "ERROR"], upper(var.log_level))
    error_message = "log_level must be one of: DEBUG, INFO, WARN, ERROR"
  }
}

variable "endpoint" {
  description = "Webhook endpoint path"
  type        = string
}

variable "llm_model" {
  description = "LLM model name"
  type        = string
}

variable "llm_cache_ttl_minutes" {
  description = "LLM cache TTL in minutes"
  type        = number
  default     = 60

  validation {
    condition     = var.llm_cache_ttl_minutes > 0
    error_message = "llm_cache_ttl_minutes must be a positive integer"
  }
}

variable "llm_timeout_seconds" {
  description = "LLM API timeout in seconds"
  type        = number
  default     = 30

  validation {
    condition     = var.llm_timeout_seconds > 0
    error_message = "llm_timeout_seconds must be a positive integer"
  }
}

variable "typing_indicator_delay_seconds" {
  description = "Delay before showing typing indicator"
  type        = number
  default     = 5

  validation {
    condition     = var.typing_indicator_delay_seconds > 0
    error_message = "typing_indicator_delay_seconds must be a positive integer"
  }
}

variable "typing_indicator_timeout_seconds" {
  description = "Typing indicator display duration (LINE API requires 5-60 seconds)"
  type        = number
  default     = 30

  validation {
    condition     = var.typing_indicator_timeout_seconds >= 5 && var.typing_indicator_timeout_seconds <= 60
    error_message = "typing_indicator_timeout_seconds must be between 5 and 60 seconds"
  }
}
