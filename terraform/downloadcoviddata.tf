locals {

  downloadcovid_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]
}

data "google_cloudfunctions_function" "downloadcovid" {
  name = "DownloadCovidDataTotal"
}

resource "google_service_account" "downloadcovid-invoker" {
  account_id   = "downloadcovid-invoker-sa"
  display_name = "DownloadCovidDataTotal invoker"
}

resource "google_project_iam_member" "downloadcovid-invoker" {
  count  = length(local.downloadcovid_roles)
  role   = local.downloadcovid_roles[count.index]
  member = "serviceAccount:${google_service_account.downloadcovid-invoker.email}"
}

resource "google_cloud_scheduler_job" "downloadcovid-worker" {
  name             = "downloadcovid-worker"
  region           = var.cloudscheduler_location
  schedule         = "0 3 * * *"
  time_zone        = "Europe/Prague"
  attempt_deadline = "600s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "GET"
    uri         = data.google_cloudfunctions_function.downloadcovid.https_trigger_url
    oidc_token {
      audience              = data.google_cloudfunctions_function.downloadcovid.https_trigger_url
      service_account_email = google_service_account.downloadcovid-invoker.email
    }
  }

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]
}