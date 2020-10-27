variable "project" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "image" {
  type    = string
  default = "runatlantis/atlantis:v0.15.1"
}

variable "instance_name" {
  description = "The desired name to assign to the deployed instance"
}

variable "zone" {
  description = "The GCP zone to deploy instances into"
  type        = string
}

variable "webhook_secret" {
  description = "GitHub webhook secret"
  type        = string
}

variable "github_token" {
  description = "GitHub API token"
  type        = string
}

variable "github_user" {
  description = "GitHub user"
  type        = string
}

variable "repo_allowlist" {
  description = "GitHub repo allow list"
  type        = string
}

variable "client_email" {
  description = "Service account email address"
  type        = string
  default     = ""
}

variable "network" {
  description = "The GCP network"
  type        = string
  default     = "default"
}

variable "cos_image_name" {
  description = "The forced COS image to use instead of latest"
  default     = "cos-stable-77-12371-89-0"
}

variable "image_port" {
  description = "The port the image exposes for HTTP requests"
  type        = number
  default     = 4141
}