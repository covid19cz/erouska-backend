variable "project" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

# The location for the app engine; this implicitly defines the region for
# scheduler jobs as specified by the cloudscheduler_location variable but the
# values are sometimes different (as in the default values) so they are kept as
# separate variables.
# https://cloud.google.com/appengine/docs/locations
variable "appengine_location" {
  type    = string
  default = "us-central"
}

# The cloudscheduler_location MUST use the same region as appengine_location but
# it must include the region number even if this is omitted from the
# appengine_location (as in the default values).
variable "cloudscheduler_location" {
  type    = string
  default = "us-central1"
}

terraform {
  required_providers {
    google      = "~> 3.32"
    google-beta = "~> 3.32"
    null        = "~> 2.1"
    random      = "~> 2.3"
  }
}
