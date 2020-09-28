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

  # DownloadKeys

  efgsdownloadkeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
  ]

  # DownloadYesterdaysKeys

  efgsdownloadyesterdayskeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
  ]

  # DownloadYesterdaysKeys - invoker

  efgsdownloadyesterdayskeys_invoker_roles = [
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

# DownloadKeys

resource "google_service_account" "efgsdownloadkeys" {
  account_id   = "efgs-download-keys"
  display_name = "EfgsDownloadKeys cloud function service account"
}

resource "google_project_iam_member" "efgsdownloadkeys" {
  count  = length(local.efgsdownloadkeys_roles)
  role   = local.efgsdownloadkeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownloadkeys.email}"
}

# DownloadYesterdaysKeys

data "google_cloudfunctions_function" "efgsdownloadyesterdayskeys" {
  name    = "EfgsDownloadYesterdaysKeys"
  project = var.project
}

resource "google_service_account" "efgsdownloadyesterdayskeys" {
  account_id   = "efgs-download-yesterdays-keys"
  display_name = "EfgsDownloadYesterdaysKeys cloud function service account"
}

resource "google_project_iam_member" "efgsdownloadyesterdayskeys" {
  count  = length(local.efgsdownloadyesterdayskeys_roles)
  role   = local.efgsdownloadyesterdayskeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownloadyesterdayskeys.email}"
}

# DownloadYesterdaysKeys - invoker

resource "google_service_account" "efgsdownloadyesterdayskeys-invoker" {
  account_id   = "efgsdownloadyesterdayskeys-invoker-sa"
  display_name = "EfgsDownloadYesterdaysKeys invoker"
}

resource "google_project_iam_member" "efgsdownloadyesterdayskeys-invoker" {
  count  = length(local.efgsdownloadyesterdayskeys_invoker_roles)
  role   = local.efgsdownloadyesterdayskeys_invoker_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownloadyesterdayskeys-invoker.email}"
}

resource "google_cloud_scheduler_job" "efgsdownloadyesterdayskeys-worker" {
  name             = "efgsdownloadyesterdayskeys-worker"
  region           = var.cloudscheduler_location
  schedule         = "0 5 * * *"
  time_zone        = "Europe/Prague"
  attempt_deadline = "600s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "GET"
    uri         = data.google_cloudfunctions_function.efgsdownloadyesterdayskeys.https_trigger_url
    oidc_token {
      audience              = data.google_cloudfunctions_function.efgsdownloadyesterdayskeys.https_trigger_url
      service_account_email = google_service_account.efgsdownloadyesterdayskeys-invoker.email
    }
  }

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]
}
