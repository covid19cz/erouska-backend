variable "name" {
  type        = string
  default     = "pgadmin"
  description = "name of the instance"
}

variable "machine_type" {
  type        = string
  default     = "f1-micro"
  description = "instance type, this variable affects memory size and vCPU count"
}

variable "zone" {
  type        = string
  default     = "europe-west1-b"
  description = "compute zone where instance is configured"
}

variable "domains" {
  type        = list(string)
  description = "list of domains for the provisioning of TLS certificates"
}