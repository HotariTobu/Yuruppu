# OpenTofu configuration for Yuruppu LINE bot
# See ADR: docs/adr/20251219-iac-tool.md

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  backend "gcs" {
    bucket = "yuruppu-tfstate"
    prefix = "tofu/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "secretmanager.googleapis.com",
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

# Artifact Registry for container images
resource "google_artifact_registry_repository" "yuruppu" {
  location      = var.region
  repository_id = "yuruppu"
  format        = "DOCKER"
  description   = "Container images for Yuruppu LINE bot"

  depends_on = [google_project_service.apis]
}

# Secret Manager secrets for LINE credentials
locals {
  secrets = ["LINE_CHANNEL_SECRET", "LINE_CHANNEL_ACCESS_TOKEN"]
}

resource "google_secret_manager_secret" "secrets" {
  for_each  = toset(local.secrets)
  secret_id = each.value

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

# Service accounts
resource "google_service_account" "cloudbuild" {
  account_id   = "yuruppu-cloudbuild"
  display_name = "Cloud Build for Yuruppu"
}

resource "google_service_account" "cloudrun" {
  account_id   = "yuruppu-cloudrun"
  display_name = "Cloud Run for Yuruppu"
}

resource "google_project_iam_member" "cloudbuild_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild.email}"
}

resource "google_project_iam_member" "cloudbuild_service_account_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.cloudbuild.email}"
}

resource "google_project_iam_member" "cloudbuild_artifact_registry" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.cloudbuild.email}"
}

resource "google_project_iam_member" "cloudbuild_logs" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.cloudbuild.email}"
}

# Cloud Build 2nd gen connection and repository are created via gcloud
# See docs/deployment.md for setup instructions
locals {
  cloudbuild_repository_id = "projects/${var.project_id}/locations/${var.region}/connections/${var.github_connection}/repositories/${var.github_repo}"
}

# Cloud Build trigger for main branch (Gen 2)
resource "google_cloudbuild_trigger" "main_push" {
  name            = "yuruppu-main-push"
  description     = "Deploy Yuruppu on push to main branch"
  location        = var.region
  service_account = google_service_account.cloudbuild.id

  repository_event_config {
    repository = local.cloudbuild_repository_id

    push {
      branch = "^main$"
    }
  }

  filename = "cloudbuild.yaml"

  substitutions = {
    _REGION       = var.region
    _SERVICE_NAME = google_cloud_run_v2_service.yuruppu.name
    _REPO_NAME    = google_artifact_registry_repository.yuruppu.repository_id
  }

  depends_on = [google_project_service.apis]
}

# Cloud Run service
resource "google_cloud_run_v2_service" "yuruppu" {
  name     = "yuruppu"
  location = var.region

  template {
    service_account = google_service_account.cloudrun.email

    containers {
      # Use placeholder for initial deployment, Cloud Build updates this
      image = "gcr.io/cloudrun/hello"

      dynamic "env" {
        for_each = local.secrets
        content {
          name = env.value
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.secrets[env.value].secret_id
              version = "latest"
            }
          }
        }
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }

    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
  }

  depends_on = [
    google_project_service.apis,
    google_artifact_registry_repository.yuruppu,
  ]
}

# Allow unauthenticated access to Cloud Run (for LINE webhook)
resource "google_cloud_run_v2_service_iam_member" "public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.yuruppu.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Grant Cloud Run service account access to secrets
resource "google_secret_manager_secret_iam_member" "cloudrun_secrets" {
  for_each  = toset(local.secrets)
  secret_id = google_secret_manager_secret.secrets[each.value].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloudrun.email}"
}
