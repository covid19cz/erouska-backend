locals {

  downloadcovid_invoker_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]

  downloadcoviddata_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.user"
  ]

  getcoviddata_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.viewer"
  ]

  changepushtoken_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.user"
  ]

  isehridactive_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.viewer"
  ]

  registerehrid_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountTokenCreator",
    "roles/datastore.user",
  ]

  registernotification_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.user",
    "roles/pubsub.publisher",
  ]

  registernotificationaftermath_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/datastore.user",
  ]
}

data "google_cloudfunctions_function" "downloadcovid" {
  name    = "DownloadCovidDataTotal"
  project = var.project
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

# cloudscheduler region has to be the same as appengine region...
# it might happen that we have scheduler calling function in different region
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

resource "google_service_account" "getcoviddata" {
  account_id   = "get-covid-data"
  display_name = "GetCovidData cloud function service account"
}

resource "google_service_account" "changepushtoken" {
  account_id   = "change-push-token"
  display_name = "ChangePushToken cloud function service account"
}

resource "google_service_account" "isehridactive" {
  account_id   = "is-ehrid-active"
  display_name = "IsEhridActive cloud function service account"
}

resource "google_service_account" "registerehrid" {
  account_id   = "register-ehrid"
  display_name = "RegisterEhrid cloud function service account"
}

resource "google_service_account" "registernotification" {
  account_id   = "register-notification"
  display_name = "RegisterNotification cloud function service account"
}

resource "google_service_account" "registernotificationaftermath" {
  account_id   = "reg-notification-aftermath"
  display_name = "RegisterNotificationAfterMath cloud function service account"
}

resource "google_project_iam_member" "downloadcoviddata" {
  count  = length(local.downloadcoviddata_roles)
  role   = local.downloadcoviddata_roles[count.index]
  member = "serviceAccount:${google_service_account.downloadcoviddata.email}"
}

resource "google_project_iam_member" "getcoviddata" {
  count  = length(local.getcoviddata_roles)
  role   = local.getcoviddata_roles[count.index]
  member = "serviceAccount:${google_service_account.getcoviddata.email}"
}

resource "google_project_iam_member" "changepushtoken" {
  count  = length(local.changepushtoken_roles)
  role   = local.changepushtoken_roles[count.index]
  member = "serviceAccount:${google_service_account.changepushtoken.email}"
}

resource "google_project_iam_member" "isehridactive" {
  count  = length(local.isehridactive_roles)
  role   = local.isehridactive_roles[count.index]
  member = "serviceAccount:${google_service_account.isehridactive.email}"
}

resource "google_project_iam_member" "registerehrid" {
  count  = length(local.registerehrid_roles)
  role   = local.registerehrid_roles[count.index]
  member = "serviceAccount:${google_service_account.registerehrid.email}"
}

resource "google_project_iam_member" "registernotification" {
  count  = length(local.registernotification_roles)
  role   = local.registernotification_roles[count.index]
  member = "serviceAccount:${google_service_account.registernotification.email}"
}

resource "google_project_iam_member" "registernotificationaftermath" {
  count  = length(local.registernotificationaftermath_roles)
  role   = local.registernotificationaftermath_roles[count.index]
  member = "serviceAccount:${google_service_account.registernotificationaftermath.email}"
}
