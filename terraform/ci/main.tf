terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-prod"
    prefix = "terraform-ci/erouska-backend/state"
  }
}

provider "google" {
  project = var.project
  region  = var.region
}

terraform {
  required_providers {
    google      = "~> 3.32"
    google-beta = "~> 3.32"
    null        = "~> 2.1"
    random      = "~> 2.3"
  }
}