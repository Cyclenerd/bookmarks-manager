# Service Agent for the Storage Transfer Service
data "google_storage_transfer_project_service_account" "default" {
  project = module.project.project_id
}

# Backup of the SQLite database bucket
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_transfer_job
resource "google_storage_transfer_job" "gcs-bookmarks-manager-database-backup" {
  # Use random name!
  # Deleting the transfer job takes long and is still present even though it has already been deleted.
  # The name is then still assigned.
  description = "Nightly backup of SQLite database bucket"
  project     = module.project.project_id

  transfer_spec {
    gcs_data_source {
      bucket_name = module.gcs-bookmarks-manager-database.name
    }
    gcs_data_sink {
      bucket_name = module.gcs-bookmarks-manager-backup.name
    }
  }

  schedule {
    schedule_start_date {
      year  = 2025
      month = 1
      day   = 1
    }
    schedule_end_date {
      year  = 2099
      month = 12
      day   = 31
    }
    start_time_of_day {
      hours   = 23
      minutes = 30
      seconds = 0
      nanos   = 0
    }
    repeat_interval = "86400s" # 24 hours
  }
}

# Backup of the favicons bucket
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_transfer_job
resource "google_storage_transfer_job" "gcs-bookmarks-manager-icons-backup" {
  # Use random name!
  # Deleting the transfer job takes long and is still present even though it has already been deleted.
  # The name is then still assigned.
  description = "Nightly backup of favicons bucket"
  project     = module.project.project_id

  transfer_spec {
    gcs_data_source {
      bucket_name = module.gcs-bookmarks-manager-icons.name
    }
    gcs_data_sink {
      bucket_name = module.gcs-bookmarks-manager-backup.name
      path        = "favicons/" # GCS sink Path argument must end with '/'
    }
  }

  schedule {
    schedule_start_date {
      year  = 2025
      month = 1
      day   = 1
    }
    schedule_end_date {
      year  = 2099
      month = 12
      day   = 31
    }
    start_time_of_day {
      hours   = 23
      minutes = 50
      seconds = 0
      nanos   = 0
    }
    repeat_interval = "86400s" # 24 hours
  }
}
