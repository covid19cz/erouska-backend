terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-prod"
    prefix = "terraform-prod/erouska-backend/state"
  }
}

module "erouska" {

  source                  = "../modules/erouska"
  project                 = var.project
  region                  = var.region
  cloudscheduler_location = var.cloudscheduler_location
  appengine_location      = var.appengine_location
}

module "stats-serving" {

  source  = "../modules/cdn-function"
  project = var.project
  region  = var.region

  name_prefix = "stats-serving"

  domains = ["stats.erouska.cz"]

  https_redirect = true

  // TODO: this should be an output from erouska module
  function_name = "DownloadMetrics"
}

provider "google" {
  project = var.project
  region  = var.region
}