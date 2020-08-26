locals {

  downloadcovid_invoker_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]

  downloadcoviddata_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.user"
  ]

  calculatecoviddata_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.user"
  ]
  getcoviddata_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.viewer"
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
  count  = length(local.downloadcovid_invoker_roles)
  role   = local.downloadcovid_invoker_roles[count.index]
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

resource "google_service_account" "downloadcoviddata" {
  account_id   = "download-covid-data-total"
  display_name = "DownloadCovidDataTotal cloud function service account"
}

resource "google_service_account" "calculatecoviddata" {
  account_id   = "calculate-covid-data-increase"
  display_name = "CalculateCovidDataIncrease cloud function service account"
}

resource "google_service_account" "getcoviddata" {
  account_id   = "get-covid-data"
  display_name = "GetCovidData cloud function service account"
}

resource "google_project_iam_member" "downloadcoviddata" {
  count  = length(local.downloadcoviddata_roles)
  role   = local.downloadcoviddata_roles[count.index]
  member = "serviceAccount:${google_service_account.downloadcoviddata.email}"
}

resource "google_project_iam_member" "calculatecoviddata" {
  count  = length(local.downloadcoviddata_roles)
  role   = local.calculatecoviddata_roles[count.index]
  member = "serviceAccount:${google_service_account.calculatecoviddata.email}"
}

resource "google_project_iam_member" "getcoviddata" {
  count  = length(local.getcoviddata_roles)
  role   = local.getcoviddata_roles[count.index]
  member = "serviceAccount:${google_service_account.getcoviddata.email}"
}
