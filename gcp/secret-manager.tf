# Secret Manager for storing GitHub App credentials
# https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/blob/v53.0.0/modules/secret-manager/README.md
module "secret-manager" {
  source     = "git::https://github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/secret-manager?ref=v53.0.0"
  project_id = module.project.project_id
  secrets = {
    bookmarks-manager-password = {
      global_replica_locations = {
        (var.region) = null
      }
      iam = {
        "roles/secretmanager.secretAccessor" = [
          module.service-account-cloud-run-bookmarks-manager.iam_email
        ]
      }
      version_config = {
        # Secret Version TTL after destruction request.
        # This is a part of the delayed delete feature on Secret Version.
        # For secret with versionDestroyTtl>0,
        # version destruction doesn't happen immediately on calling destroy
        # instead the version goes to a disabled state and
        # the actual destruction happens after this TTL expires.
        destroy_ttl = "172800s" # 2 days
      }
    }
  }
  depends_on = [
    time_sleep.wait_for_service_account_cloud_run
  ]
}

# Create initial placeholder secret versions (will be updated with actual values)
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_version
resource "google_secret_manager_secret_version" "secret-version-bookmarks-manager-password" {
  secret      = module.secret-manager.ids["bookmarks-manager-password"]
  secret_data = var.password
}
