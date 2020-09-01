locals {

  cloudbuild_roles = [
    "roles/cloudfunctions.developer",
    "roles/iam.serviceAccountUser"
  ]

}

data "google_project" "project" {
  project_id = var.project
}

resource "google_project_service" "services" {
  for_each = toset([
    "cloudbuild.googleapis.com",
    "cloudscheduler.googleapis.com",
    "firebase.googleapis.com",
    "iam.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

resource "google_project_iam_member" "cloudbuild" {
  count  = length(local.cloudbuild_roles)
  role   = local.cloudbuild_roles[count.index]
  member = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"

  depends_on = [
    google_project_service.services["cloudbuild.googleapis.com"],
  ]
}


# Cloud Scheduler requires AppEngine projects!
#
# If your project already has GAE enabled, run `terraform import google_app_engine_application.app $PROJECT_ID`
resource "google_app_engine_application" "app" {
  location_id = var.appengine_location
}
