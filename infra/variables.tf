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

variable "llm_model" {
  description = "LLM model name (e.g., gemini-2.5-flash-lite)"
  type        = string
}
