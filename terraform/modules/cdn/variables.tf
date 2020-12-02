variable "project" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "bucket_name" {
  type        = string
  description = "GCS bucket name for backend bucket serving"
}

variable "name_prefix" {
  type        = string
  default     = "en"
  description = "name prefix for resources created"
}

variable "domains" {
  default     = []
  description = "domains for TLS certificate"
}

variable "https_redirect" {
  description = "Set to `true` to enable https redirect on the lb."
  type        = bool
  default     = false
}

terraform {
  required_providers {
    google      = "~> 3.32"
    google-beta = "~> 3.32"
    null        = "~> 2.1"
    random      = "~> 2.3"
  }
}
