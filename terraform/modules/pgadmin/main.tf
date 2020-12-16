resource "google_compute_global_address" "this" {
  name = "${var.name}-lb"
}

resource "google_compute_instance" "this" {
  name         = var.name
  machine_type = var.machine_type
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "ubuntu-1804-bionic-v20201211a"
    }
  }

  tags = ["${var.name}-http-hc"]

  network_interface {
    network = "default"

    access_config {
      // Ephemeral IP
    }
  }

  metadata_startup_script = ""

  service_account {
    scopes = ["userinfo-email", "compute-ro", "storage-ro"]
  }
}

resource "google_compute_instance_group" "this" {
  name = "${var.name}-servers"
  zone = var.zone
  instances = [
    google_compute_instance.this.id,
  ]

  named_port {
    name = "http"
    port = "80"
  }

  named_port {
    name = "https"
    port = "443"
  }
}

resource "google_compute_firewall" "hc-http" {
  name    = "${var.name}-hc-http"
  network = "default"
  allow {
    protocol = "tcp"
    ports = [
      80,
    ]
  }
  source_ranges = [
    "35.191.0.0/16",
    "209.85.152.0/22",
    "209.85.204.0/22",
    "130.211.0.0/22",
  ]
  target_tags = [
    "${var.name}-http-hc",
  ]
}

resource "google_compute_http_health_check" "this" {
  name               = "${var.name}-health-check"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}

resource "google_compute_backend_service" "this" {
  name = "${var.name}-backend"

  protocol    = "HTTP"
  port_name   = "http"
  timeout_sec = 30
  health_checks = [
    google_compute_http_health_check.this.id,
  ]

  backend {
    group = google_compute_instance_group.this.id
  }
}

resource "google_compute_url_map" "this" {
  name            = "${var.name}-urlmap"
  default_service = google_compute_backend_service.this.id
}

resource "google_compute_target_http_proxy" "http" {
  name    = "${var.name}-http-proxy"
  url_map = google_compute_url_map.this.id
}

resource "google_compute_target_https_proxy" "https" {
  name             = "${var.name}-https-proxy"
  url_map          = google_compute_url_map.this.id
  ssl_certificates = [google_compute_managed_ssl_certificate.this.id]
}

resource "google_compute_global_forwarding_rule" "http" {
  name       = "${var.name}-http"
  target     = google_compute_target_http_proxy.http.id
  port_range = "80"
  ip_address = google_compute_global_address.this.address
}

resource "google_compute_global_forwarding_rule" "https" {
  name       = "${var.name}-https"
  target     = google_compute_target_https_proxy.https.id
  port_range = "443"
  ip_address = google_compute_global_address.this.address
}

resource "google_compute_managed_ssl_certificate" "this" {
  name = var.name

  managed {
    domains = var.domains
  }
}