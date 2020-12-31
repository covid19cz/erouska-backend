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

module "atlantis" {
  source = "git::https://github.com/pipetail/terraform-atlantis-gce.git?ref=v0.3.2"

  region  = var.region
  project = var.project

  zone           = var.zone
  instance_name  = var.instance_name
  instance_type  = var.instance_type
  repo_allowlist = var.repo_allowlist
  image          = var.image
  webhook_secret = var.webhook_secret
  github_token   = var.github_token
  github_user    = var.github_user

}

output "atlantis_ip" {
  value = module.atlantis.atlantis_ip
}