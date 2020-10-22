terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-prod"
    prefix = "terraform-dev/erouska-backend/state"
  }
}

module "erouska" {

  source                  = "../modules/erouska"
  project                 = var.project
  region                  = var.region
  cloudscheduler_location = var.cloudscheduler_location
  appengine_location      = var.appengine_location
}


resource "null_resource" "exdawample" {}

provider "google" {
  project = var.project
  region  = var.region
}