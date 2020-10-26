locals {
  instance_name = format("%s-%s", var.instance_name, substr(md5(module.gce-container.container.image), 0, 8))
  target_tags   = ["ci", "atlantis"]
}

resource "google_compute_instance" "atlantis" {
  project      = var.project
  name         = local.instance_name
  machine_type = "e2-micro"
  zone         = var.zone

  allow_stopping_for_update = true

  boot_disk {
    initialize_params {
      image = module.gce-container.source_image
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.static.address
    }
  }

  tags = local.target_tags

  metadata = {
    gce-container-declaration = module.gce-container.metadata_value
    google-logging-enabled    = "true"
    google-monitoring-enabled = "true"
  }

  labels = {
    container-vm = module.gce-container.vm_container_label
  }

  service_account {
    email = google_service_account.atlantis.email
    scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]
  }
}

module "gce-container" {
  source = "github.com/terraform-google-modules/terraform-google-container-vm"
  #version = "0.1.0"

  container = {
    image = "runatlantis/atlantis:v0.15.1"
    args = [
      "server",
      "--atlantis-url=http://${google_compute_address.static.address}",
      "--gh-user=${var.github_user}",
      "--gh-token=${var.github_token}",
      "--gh-webhook-secret=${var.webhook_secret}",
      "--repo-allowlist=${var.repo_allowlist}",
    ]
  }

  restart_policy = "Always"
}

resource "google_compute_firewall" "ingress-to-instance" {
  name    = "atlantis"
  project = var.project
  network = var.network
  allow {
    protocol = "tcp"
    ports = [
      var.image_port
    ]
  }
  source_ranges = ["0.0.0.0/0"]
  target_tags   = local.target_tags
}

resource "google_compute_address" "static" {
  name = "atlantis-ipv4"
}

resource "google_service_account" "atlantis" {
  account_id   = "atlantis"
  display_name = "atlantis"
}

output "atlantis_ip" {
  description = "The public IP address of the deployed instance"
  value       = google_compute_instance.atlantis.network_interface.0.access_config.0.nat_ip
}