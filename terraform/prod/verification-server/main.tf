terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-verification-prod"
    prefix = "terraform-prod/exposure-notification-verification-server/state"
  }
}

module "vf" {
  source = "git::https://github.com/google/exposure-notifications-verification-server.git//terraform?ref=v0.9.0"

  project = var.project
  region  = var.region

  database_tier         = var.database_tier
  database_disk_size_gb = var.database_disk_size_gb

  cloudscheduler_location = var.cloudscheduler_location
  appengine_location      = var.appengine_location

  cloudrun_location = var.cloudrun_location


  database_max_connections = var.database_max_connections
  database_backup_location = var.database_backup_location


  redis_location             = var.redis_location
  redis_alternative_location = var.redis_alternative_location
  redis_cache_size           = var.redis_cache_size

  service_environment = var.service_environment

}

module "alerting" {
  source             = "git::https://github.com/google/exposure-notifications-verification-server.git//terraform/alerting?ref=v0.9.0"
  project            = var.project
  notification-email = var.notification-email
  server-host        = replace(module.vf.server_url, "https://", "")
  apiserver-host     = replace(module.vf.apiserver_url, "https://", "")
  adminapi-host      = replace(module.vf.adminapi_url, "https://", "")
}

provider "google" {
  project = var.project
  region  = var.region

  user_project_override = true
}

provider "google-beta" {
  project = var.project
  region  = var.region

  user_project_override = true
}

output "vf" {
  value = module.vf
}

output "alerting" {
  value = module.alerting
}