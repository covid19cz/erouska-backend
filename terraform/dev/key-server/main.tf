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
  source = "git::https://github.com/google/exposure-notifications-server.git//terraform?ref=v0.22.1"

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
  db_user                  = var.db_user
  cloudsql_tier            = var.cloudsql_tier
  cloudsql_disk_size_gb    = var.cloudsql_disk_size_gb
  generate_cron_schedule   = var.generate_cron_schedule
  cloudsql_max_connections = var.cloudsql_max_connections
  cloudsql_backup_location = var.cloudsql_backup_location
  db_version               = var.db_version

  service_environment = {
    jwks = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    generate = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    federationout = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    federationin = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    exposure = {
      OBSERVABILITY_EXPORTER       = "NOOP"
      PROJECT_ID                   = var.project
      MAX_KEYS_ON_PUBLISH          = 50
      REVISION_TOKEN_KEY_ID        = "projects/erouska-key-server-dev/locations/europe-west1/keyRings/revision-tokens/cryptoKeys/token-key"
      MAX_SAME_START_INTERVAL_KEYS = 15
    }
    export = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    cleanup_exposure = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    cleanup_export = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
    key-rotation = {
      OBSERVABILITY_EXPORTER = "NOOP"
      PROJECT_ID             = var.project
    }
  }
}

module "cdn" {

  source  = "../../modules/cdn-bucket"
  project = var.project
  region  = var.region

  name_prefix = "exposure-keys"

  // TODO: this output is supported in newer terraform module version
  //bucket_name = module.en.export_bucket
  bucket_name = "exposure-notification-export-ejjud"
}

module "pgadmin" {
  source  = "../../modules/pgadmin"
  zone    = "${var.region}-b"
  domains = var.pgadmin_domains
}

output "en" {
  value = module.en
}
