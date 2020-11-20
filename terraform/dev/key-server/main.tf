terraform {
  backend "gcs" {
    bucket = "terraform-erouska-key-server-dev"
    prefix = "exposure-notification-server/state"
  }
}

provider "google" {
  project = var.project
  region  = var.region
}

module "en" {
  source = "git::https://github.com/google/exposure-notifications-server.git//terraform?ref=v0.9.2"

  project = var.project
  region  = var.region

  appengine_location       = var.appengine_location
  storage_location         = var.storage_location
  cloudrun_location        = var.cloudrun_location
  cloudscheduler_location  = var.cloudscheduler_location
  kms_location             = var.kms_location
  network_location         = var.network_location
  db_location              = var.db_location
  db_name                  = var.db_name
  cloudsql_tier            = var.cloudsql_tier
  cloudsql_disk_size_gb    = var.cloudsql_disk_size_gb
  generate_cron_schedule   = var.generate_cron_schedule
  cloudsql_max_connections = var.cloudsql_max_connections
  cloudsql_backup_location = var.cloudsql_backup_location
}

module "cdn" {

  source  = "../../modules/cdn"
  project = var.project
  region  = var.region

  name_prefix = "exposure-keys"

  // TODO: this output is supported in newer terraform module version
  //bucket_name = module.en.export_bucket
  bucket_name = "exposure-notification-export-ejjud"
}


output "en" {
  value = module.en
}
