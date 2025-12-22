# Output values for Yuruppu infrastructure

output "artifact_registry_url" {
  description = "Artifact Registry URL for container images"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.yuruppu.repository_id}"
}

output "cloud_run_url" {
  description = "Cloud Run service URL"
  value       = try(google_cloud_run_v2_service.yuruppu.uri, null)
}

output "webhook_url" {
  description = "LINE webhook URL"
  value       = try("${google_cloud_run_v2_service.yuruppu.uri}/webhook", null)
}
