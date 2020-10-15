terraform {
  backend "gcs" {
    bucket = "erouska-terraform-state-prod"
    prefix = "terraform-dev/exposure-notification-verification-server/alerting/state"
  }
}

module "alerting" {
  source             = "git::https://github.com/google/exposure-notifications-verification-server.git//terraform/alerting?ref=v0.9.0"
  project            = var.project
  notification-email = var.notification-email
  server-host        = var.server-host
  apiserver-host     = var.apiserver-host
  adminapi-host      = var.adminapi-host
}

output "alerting" {
  value = module.alerting
}