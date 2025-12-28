# Google Cloud APIs to enable for the project
variable "apis" {
  description = "List of Google Cloud APIs to be enable"
  type        = list(string)
  default = [
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "compute.googleapis.com",
    "iam.googleapis.com",
    "logging.googleapis.com",
    "orgpolicy.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "storage.googleapis.com",
  ]
}

# Google Cloud project ID where resources will be created
variable "project_id" {
  description = "Existing Google Cloud project ID"
  type        = string
  nullable    = false

  # https://cloud.google.com/resource-manager/docs/creating-managing-projects#before_you_begin
  validation {
    # Must be 6 to 30 characters in length.
    # Can only contain lowercase letters, numbers, and hyphens.
    # Must start with a letter.
    # Cannot end with a hyphen.
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Invalid Google Cloud project ID!"
  }
}

# Google Cloud region for deploying resources
variable "region" {
  description = "Google Cloud region name"
  type        = string
  default     = "us-central1"
  nullable    = false

  validation {
    condition     = can(regex("^[a-z][-a-z]+[0-9]$", var.region))
    error_message = "Invalid Google Cloud region name!"
  }
}

# Google Cloud region for backups
variable "backup_region" {
  description = "Google Cloud region name for backup"
  type        = string
  default     = "us-east1"
  nullable    = false

  validation {
    condition     = can(regex("^[a-z][-a-z]+[0-9]$", var.backup_region))
    error_message = "Invalid Google Cloud region name!"
  }

  validation {
    condition     = var.backup_region != var.region
    error_message = "The backup region must be different from the primary region!"
  }
}

variable "username" {
  description = "Username for HTTP Basic Auth"
  type        = string
  sensitive   = true

  validation {
    condition     = can(regex("^[a-zA-Z0-9]+$", var.username))
    error_message = "The username must only contain alphanumeric characters."
  }
}

variable "password" {
  description = "Password for HTTP Basic Auth"
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.password) >= 8
    error_message = "The password must be at least 8 characters long."
  }
}
