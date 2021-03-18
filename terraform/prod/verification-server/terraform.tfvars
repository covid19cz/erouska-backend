database_tier              = "db-custom-1-3840"
database_disk_size_gb      = "16"
region                     = "europe-west1"
cloudscheduler_location    = "europe-west1"
appengine_location         = "europe-west"
redis_location             = "europe-west1-b"
redis_alternative_location = "europe-west1-c"
database_max_connections   = "10000"
database_backup_location   = "eu"
redis_cache_size           = 1
service_environment = {
  apiserver = {
    CERTIFICATE_ISSUER     = "cz.covid19cz.erouska"
    CERTIFICATE_AUDIENCE   = "covid19cz"
    OBSERVABILITY_EXPORTER = "NOOP"
    RATE_LIMIT_TOKENS      = "150"
    RATE_LIMIT_INTERVAL    = "1m"
  }
  server = {
    FIREBASE_PRIVACY_POLICY_URL   = "TODO"
    FIREBASE_TERMS_OF_SERVICE_URL = "TODO"
    OBSERVABILITY_EXPORTER        = "NOOP"
  }
  modeler = {
    OBSERVABILITY_EXPORTER = "NOOP"
  }
  e2e-runner = {
    OBSERVABILITY_EXPORTER = "NOOP"
  }
  enx-redirect = {
    OBSERVABILITY_EXPORTER = "NOOP"
  }
  cleanup = {
    OBSERVABILITY_EXPORTER = "NOOP"
  }
  adminapi = {
    OBSERVABILITY_EXPORTER = "NOOP"
    RATE_LIMIT_TOKENS      = "150"
    RATE_LIMIT_INTERVAL    = "1m"
  }
  stats-puller = {
    KEY_SERVER_URL = "https://exposure-fghz64a2xa-ew.a.run.app"
  }
}
notification-email = "marek.bartik@pipetail.io"
project            = "cosmic-bonus-287409"
pgadmin_domains    = ["pgadmin.verification.prod.evec.cz"]

