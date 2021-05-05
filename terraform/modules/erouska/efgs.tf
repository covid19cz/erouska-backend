locals {
  # UploadKeys

  efgsuploadkeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
    "roles/cloudsql.editor"
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
    "roles/redis.editor",
    "roles/pubsub.publisher",
  ]

  # DownloadKeys - invoker

  efgsdownkeys_invoker_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]

  # DownloadYesterdaysKeys

  efgsdownyestkeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
    "roles/pubsub.publisher",
  ]

  # DownloadYesterdaysKeys - invoker

  efgsdownyestkeys_invoker_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]

  # ImportKeys

  efgsimportkeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
  ]

  # RemoveOldKeys

  efgsremoveoldkeys_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
    "roles/cloudsql.editor",
  ]

  # RemoveOldKeys - invoker

  efgsremoveoldkeys_invoker_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/iam.serviceAccountUser"
  ]

  # EfgsIssueTestingVerificationCode

  issuetestingverificationcode_roles = [
    "roles/cloudfunctions.serviceAgent",
    "roles/secretmanager.secretAccessor",
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
  count = (data.google_cloudfunctions_function.efgsuploadkeys.https_trigger_url != null) ? 1 : 0

  name             = "efgsuploadkeys-worker"
  region           = var.cloudscheduler_location
  schedule         = "*/15 * * * *"
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

data "google_cloudfunctions_function" "efgsdownkeys" {
  name    = "EfgsDownloadKeys"
  project = var.project
}

resource "google_service_account" "efgsdownloadkeys" {
  account_id   = "efgs-download-keys"
  display_name = "EfgsDownloadKeys cloud function service account"
}

resource "google_project_iam_member" "efgsdownloadkeys" {
  count  = length(local.efgsdownloadkeys_roles)
  role   = local.efgsdownloadkeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownloadkeys.email}"
}

# DownloadKeys - invoker

resource "google_service_account" "efgsdownkeys-invoker" {
  account_id   = "efgsdownkeys-invoker-sa"
  display_name = "EfgsDownloadKeys invoker"
}

resource "google_project_iam_member" "efgsdownkeys-invoker" {
  count  = length(local.efgsdownkeys_invoker_roles)
  role   = local.efgsdownkeys_invoker_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownkeys-invoker.email}"
}

resource "google_cloud_scheduler_job" "efgsdownkeys-worker" {
  count = (data.google_cloudfunctions_function.efgsdownkeys.https_trigger_url != null) ? 1 : 0

  name             = "efgsdownkeys-worker"
  region           = var.cloudscheduler_location
  schedule         = "*/15 * * * *"
  time_zone        = "Europe/Prague"
  attempt_deadline = "600s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "GET"
    uri         = data.google_cloudfunctions_function.efgsdownkeys.https_trigger_url
    oidc_token {
      audience              = data.google_cloudfunctions_function.efgsdownkeys.https_trigger_url
      service_account_email = google_service_account.efgsdownkeys-invoker.email
    }
  }

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]
}

# DownloadYesterdaysKeys

data "google_cloudfunctions_function" "efgsdownyestkeys" {
  name    = "EfgsDownloadYesterdaysKeys"
  project = var.project
}

resource "google_service_account" "efgsdownyestkeys" {
  account_id   = "efgs-download-yesterdays-keys"
  display_name = "EfgsDownloadYesterdaysKeys cloud function service account"
}

resource "google_project_iam_member" "efgsdownyestkeys" {
  count  = length(local.efgsdownyestkeys_roles)
  role   = local.efgsdownyestkeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownyestkeys.email}"
}

# DownloadYesterdaysKeys - invoker

resource "google_service_account" "efgsdownyestkeys-invoker" {
  account_id   = "efgsdownyestkeys-invoker-sa"
  display_name = "EfgsDownloadYesterdaysKeys invoker"
}

resource "google_project_iam_member" "efgsdownyestkeys-invoker" {
  count  = length(local.efgsdownyestkeys_invoker_roles)
  role   = local.efgsdownyestkeys_invoker_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsdownyestkeys-invoker.email}"
}

resource "google_cloud_scheduler_job" "efgsdownyestkeys-worker" {
  count = (data.google_cloudfunctions_function.efgsdownyestkeys.https_trigger_url != null) ? 1 : 0

  name             = "efgsdownyestkeys-worker"
  region           = var.cloudscheduler_location
  schedule         = "0 2 * * *"
  time_zone        = "Europe/Prague"
  attempt_deadline = "600s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "GET"
    uri         = data.google_cloudfunctions_function.efgsdownyestkeys.https_trigger_url
    oidc_token {
      audience              = data.google_cloudfunctions_function.efgsdownyestkeys.https_trigger_url
      service_account_email = google_service_account.efgsdownyestkeys-invoker.email
    }
  }

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]
}

# ImportKeys

data "google_cloudfunctions_function" "efgsimportkeys" {
  name    = "EfgsImportKeys"
  project = var.project
}

resource "google_service_account" "efgsimportkeys" {
  account_id   = "efgs-import-keys"
  display_name = "EfgsImportKeys cloud function service account"
}

resource "google_project_iam_member" "efgsimportkeys" {
  count  = length(local.efgsimportkeys_roles)
  role   = local.efgsimportkeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsimportkeys.email}"
}

# RemoveOldKeys

data "google_cloudfunctions_function" "efgsremoveoldkeys" {
  name    = "EfgsRemoveOldKeys"
  project = var.project
}

resource "google_service_account" "efgsremoveoldkeys" {
  account_id   = "efgs-remove-old-keys"
  display_name = "EfgsRemoveOldKeys cloud function service account"
}

resource "google_project_iam_member" "efgsremoveoldkeys" {
  count  = length(local.efgsremoveoldkeys_roles)
  role   = local.efgsremoveoldkeys_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsremoveoldkeys.email}"
}

# RemoveOldKeys - invoker

resource "google_service_account" "efgsremoveoldkeys-invoker" {
  account_id   = "efgsremoveoldkeys-invoker-sa"
  display_name = "EfgsRemoveOldKeys invoker"
}

resource "google_project_iam_member" "efgsremoveoldkeys-invoker" {
  count  = length(local.efgsremoveoldkeys_invoker_roles)
  role   = local.efgsremoveoldkeys_invoker_roles[count.index]
  member = "serviceAccount:${google_service_account.efgsremoveoldkeys-invoker.email}"
}

resource "google_cloud_scheduler_job" "efgsremoveoldkeys-worker" {
  count = (data.google_cloudfunctions_function.efgsremoveoldkeys.https_trigger_url != null) ? 1 : 0

  name             = "efgsremoveoldkeys-worker"
  region           = var.cloudscheduler_location
  schedule         = "0 6 * * *"
  time_zone        = "Europe/Prague"
  attempt_deadline = "600s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "GET"
    uri         = data.google_cloudfunctions_function.efgsremoveoldkeys.https_trigger_url
    oidc_token {
      audience              = data.google_cloudfunctions_function.efgsremoveoldkeys.https_trigger_url
      service_account_email = google_service_account.efgsremoveoldkeys-invoker.email
    }
  }

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]
}

# IssueTestingVerificationCode

data "google_cloudfunctions_function" "issuetestingverificationcode" {
  name    = "EfgsIssueTestingVerificationCode"
  project = var.project
}

resource "google_service_account" "issuetestingverificationcode" {
  account_id   = "efgs-issue-tst-verif-code"
  display_name = "EfgsIssueTestingVerificationCode cloud function service account"
}

resource "google_project_iam_member" "issuetestingverificationcode" {
  count  = length(local.issuetestingverificationcode_roles)
  role   = local.issuetestingverificationcode_roles[count.index]
  member = "serviceAccount:${google_service_account.issuetestingverificationcode.email}"
}
