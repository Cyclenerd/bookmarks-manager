# https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/blob/v49.1.0/modules/gcs/README.md

# GCS bucket for storing Terraform state
module "gcs-bookmarks-manager-iac" {
  source        = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/gcs?ref=v49.1.0"
  project_id    = module.project.project_id
  prefix        = module.project.project_id
  name          = "bm-iac-${local.region_shortnames[var.region]}"
  location      = var.region
  versioning    = true
  force_destroy = true
  lifecycle_rules = {
    lr-0 = {
      action = {
        type = "Delete"
      }
      condition = {
        num_newer_versions = 7
      }
    }
  }
}

# GCS bucket for Cloud Build source staging
module "gcs-bookmarks-manager-cloud-build" {
  source        = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/gcs?ref=v49.1.0"
  project_id    = module.project.project_id
  prefix        = module.project.project_id
  name          = "bm-build-${local.region_shortnames[var.region]}"
  location      = var.region
  versioning    = false
  force_destroy = true
  lifecycle_rules = {
    lr-0 = {
      action = {
        type = "Delete"
      }
      condition = {
        age        = 2
        with_state = "ANY"
      }
    }
  }
  iam = {
    "roles/storage.objectAdmin" = [
      module.service-account-cloud-build.iam_email
    ]
  }
  depends_on = [
    time_sleep.wait_for_service_account_cloud_build
  ]
}

# GCS bucket for storing the SQLite database
module "gcs-bookmarks-manager-database" {
  source        = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/gcs?ref=v49.1.0"
  project_id    = module.project.project_id
  prefix        = module.project.project_id
  name          = "bm-db-${local.region_shortnames[var.region]}"
  location      = var.region
  versioning    = false
  force_destroy = true
  iam = {
    "roles/storage.objectAdmin" = [
      module.service-account-cloud-run-bookmarks-manager.iam_email
    ]
    "roles/storage.bucketViewer" = [
      "serviceAccount:${data.google_storage_transfer_project_service_account.default.email}"
    ]
    "roles/storage.objectViewer" = [
      "serviceAccount:${data.google_storage_transfer_project_service_account.default.email}"
    ]
  }
  depends_on = [
    time_sleep.wait_for_service_account_cloud_run
  ]
}

# GCS bucket for storing the favicons
module "gcs-bookmarks-manager-icons" {
  source        = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/gcs?ref=v50.0.0"
  project_id    = module.project.project_id
  prefix        = module.project.project_id
  name          = "bm-icons-${local.region_shortnames[var.region]}"
  location      = var.region
  versioning    = false
  force_destroy = true
  iam = {
    "roles/storage.objectAdmin" = [
      module.service-account-cloud-run-bookmarks-manager.iam_email
    ]
    "roles/storage.bucketViewer" = [
      "serviceAccount:${data.google_storage_transfer_project_service_account.default.email}"
    ]
    "roles/storage.objectViewer" = [
      "serviceAccount:${data.google_storage_transfer_project_service_account.default.email}"
    ]
  }
  depends_on = [
    time_sleep.wait_for_service_account_cloud_run
  ]
}

# GCS bucket for backing up the SQLite database and favicons
module "gcs-bookmarks-manager-backup" {
  source        = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/gcs?ref=v49.1.0"
  project_id    = module.project.project_id
  prefix        = module.project.project_id
  name          = "bm-backup-${local.region_shortnames[var.backup_region]}"
  location      = var.backup_region
  versioning    = true
  force_destroy = true
  lifecycle_rules = {
    lr-0 = {
      action = {
        type = "Delete"
      }
      condition = {
        num_newer_versions = 14
      }
    }
  }
  iam = {
    "roles/storage.admin" = [
      "serviceAccount:${data.google_storage_transfer_project_service_account.default.email}"
    ]
  }
}
