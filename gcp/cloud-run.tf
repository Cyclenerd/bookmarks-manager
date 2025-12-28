# Get the container image from Artifact Registry
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/artifact_registry_docker_image
data "google_artifact_registry_docker_image" "container-image-bookmarks-manager" {
  project       = module.project.project_id
  location      = var.region
  repository_id = module.artifact-registry-container.name
  image_name    = "app:latest" # Defined in cloudbuild-container.template.yaml
  depends_on = [
    null_resource.build-github-runners-manager-container
  ]
}

# Deploy the Bookmarks Manager service on Cloud Run
# https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/blob/v49.1.0/modules/cloud-run-v2/README.md
module "cloud_run_github_runners_manager" {
  source     = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/cloud-run-v2?ref=v50.0.0"
  project_id = module.project.project_id
  name       = "bookmarks-manager-${local.region_shortnames[var.region]}"
  type       = "SERVICE"
  region     = var.region
  service_config = {
    # Second generation Cloud Run for faster CPU and bucket mount
    gen2_execution_environment = true
    scaling = {
      min_instance_count = 0
      max_instance_count = 1
    }
    timeout = "300s" # 5min, Warning: Import from Firefox may take longer. The import fails.
  }
  containers = {
    bookmarks-manager = {
      image = data.google_artifact_registry_docker_image.container-image-bookmarks-manager.self_link
      resources = {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
        cpu_idle          = true # Charged only when processing requests. CPU is limited outside of requests.
        startup_cpu_boost = true # Start containers faster by allocating more CPU during startup time.
      }
      env = {
        HTTP_AUTH_USERNAME = var.username
        DATABASE_PATH      = "/var/lib/bookmarks/bookmarks.db"
      }
      env_from_key = {
        HTTP_AUTH_PASSWORD = {
          # Secret may only be {secret} or
          # projects/{project}/secrets/{secret},
          # and secret may only have alphanumeric characters, hyphens, and underscores.
          # Secret must be a global secret not a regional secret!
          secret  = module.secret-manager.ids["bookmarks-manager-password"]
          version = "latest"
        }
      }
      volume_mounts = {
        "database" = "/var/lib/bookmarks"
        "icons"    = "/web/app/static/favicons"
      }
    }
  }
  volumes = {
    database = {
      gcs = {
        bucket       = module.gcs-bookmarks-manager-database.name
        is_read_only = false
      }
    }
    icons = {
      gcs = {
        bucket       = module.gcs-bookmarks-manager-icons.name
        is_read_only = false
      }
    }
  }
  service_account_config = {
    create = false
    email  = module.service-account-cloud-run-bookmarks-manager.email
  }
  iam = {
    "roles/run.invoker" = ["allUsers"] # Public
  }
  deletion_protection = false
  depends_on = [
    time_sleep.wait_for_service_account_cloud_run
  ]
}
