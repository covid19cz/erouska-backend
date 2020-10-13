locals {

  # UploadKeys

  efgsuploadkeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor"
  ]

  # UploadKeys - invoker

  efgsuploadkeys_invoker_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]
}

# UploadKeys

data "google_cloudfunctions_function" "efgsuploadkeys" {
  name    = "EfgsUploadKeys"
  project = var.project
}

resource "google_service_account" "efgsuploadkeys" {
  account_id   = "efgs-upload-keys"
  display_name = "EfgsUploadKeys cloud function service account"
}

resource "google_project_iam_member" "efgsuploadkeys" {
  count  = length(local.efgsuploadkeys_roles)
  role   = local.efgsuploadkeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsuploadkeys.email}"
}

# UploadKeys - invoker

resource "google_service_account" "efgsuploadkeys-invoker" {
  account_id   = "efgsuploadkeys-invoker-sa"
  display_name = "EfgsUploadKeys invoker"
}

resource "google_project_iam_member" "efgsuploadkeys-invoker" {
  count  = length(local.efgsuploadkeys_invoker_roles)
  role   = local.efgsuploadkeys_invoker_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsuploadkeys-invoker.email}"
}

resource "google_cloud_scheduler_job" "efgsuploadkeys-worker" {
  name             = "efgsuploadkeys-worker"
  region           = var.cloudscheduler_location
  schedule         = "0 */2 * * *"
  time_zone        = "Europe/Prague"
  attempt_deadline = "600s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "GET"
    uri         = data.google_cloudfunctions_function.efgsuploadkeys.https_trigger_url
    oidc_token {
      audience              = data.google_cloudfunctions_function.efgsuploadkeys.https_trigger_url
      service_account_email = google_service_account.efgsuploadkeys-invoker.email
    }
  }

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]
}
