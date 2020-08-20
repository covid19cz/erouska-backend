data "google_project" "project" {
  project_id = var.project
}

provider "google" {
  project = var.project
  region  = var.region
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

resource "google_project_iam_member" "cloudbuild-deploy" {
  role   = "roles/run.admin"
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
