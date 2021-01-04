terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-keyserver-prod"
    prefix = "terraform-prod/exposure-notification-server/state"
  }
}

module "en" {
  source = "git::https://github.com/google/exposure-notifications-server.git//terraform?ref=d51abd8"

  project = var.project
  region  = var.region

  appengine_location       = var.appengine_location
  storage_location         = var.storage_location
  cloudrun_location        = var.cloudrun_location
  cloudscheduler_location  = var.cloudscheduler_location
  kms_location             = var.kms_location
  network_location         = var.network_location
  db_location              = var.db_location
  db_user                  = var.db_user
  db_name                  = var.db_name
  cloudsql_tier            = var.cloudsql_tier
  cloudsql_disk_size_gb    = var.cloudsql_disk_size_gb
  generate_cron_schedule   = var.generate_cron_schedule
  cloudsql_max_connections = var.cloudsql_max_connections
  cloudsql_backup_location = var.cloudsql_backup_location
  db_version               = var.db_version

  service_environment = {
    jwks = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    generate = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    federationout = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    federationin = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    exposure = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    export = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    cleanup_exposure = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    cleanup_export = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
    key_rotation = {
      OBSERVABILITY_EXPORTER = "NOOP"
    }
  }
}

module "cdn" {

  source  = "../../modules/cdn-bucket"
  project = var.project
  region  = var.region

  name_prefix = "exposure-keys"

  domains = ["cdn.erouska.cz"]

  https_redirect = true

  // TODO: this output is supported in newer terraform module version
  //bucket_name = module.en.export_bucket
  bucket_name = "exposure-notification-export-qhqcx"
}

provider "google" {
  project = var.project
  region  = var.region
}

provider "google-beta" {
  project = var.project
  region  = var.region
}

output "en" {
  value = module.en
}