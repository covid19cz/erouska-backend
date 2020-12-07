terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-verification-prod"
    prefix = "terraform-prod/exposure-notification-verification-server/state"
  }
}

module "vf" {
  source = "git::https://github.com/google/exposure-notifications-verification-server.git//terraform?ref=v0.17.0"

  project = var.project
  region  = var.region

  database_tier         = var.database_tier
  database_disk_size_gb = var.database_disk_size_gb
  database_version      = "POSTGRES_12"

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
  source                      = "git::https://github.com/google/exposure-notifications-verification-server.git//terraform/alerting?ref=v0.17.0"
  verification-server-project = var.project
  monitoring-host-project     = var.project
  server_hosts                = module.vf.server_urls
  apiserver_hosts             = module.vf.apiserver_urls
  adminapi_hosts              = module.vf.adminapi_urls
  alert-notification-channels = {
    email = {
      labels = {
        email_address = var.notification-email
      }
    }
  }
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