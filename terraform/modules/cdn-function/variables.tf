variable "project" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "function_name" {
  type        = string
  description = "Google Cloud Function name for NEG endpoint"
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
    null        = "~> 3.0"
    random      = "~> 2.3"
  }
}
