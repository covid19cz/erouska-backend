resource "google_compute_global_address" "main" {
  name = "${var.name_prefix}-ip"
}

resource "google_compute_global_forwarding_rule" "http" {
  name       = "${var.name_prefix}-http-lb"
  target     = google_compute_target_http_proxy.http.self_link
  ip_address = google_compute_global_address.main.address
  port_range = "80"
}

resource "google_compute_global_forwarding_rule" "https" {
  name       = "${var.name_prefix}-https-lb"
  target     = google_compute_target_https_proxy.https.self_link
  ip_address = google_compute_global_address.main.address
  port_range = "443"
}

resource "google_compute_target_http_proxy" "http" {
  name    = "${var.name_prefix}-serving-target-proxy"
  url_map = google_compute_url_map.main.id
}

resource "google_compute_target_https_proxy" "https" {
  name    = "${var.name_prefix}-serving-target-proxy"
  url_map = google_compute_url_map.main.id

  ssl_certificates = [google_compute_managed_ssl_certificate.main.self_link]
}

resource "google_compute_url_map" "main" {
  name            = "${var.name_prefix}-serving"
  default_service = google_compute_backend_bucket.exposure_keys_serving.self_link
}

resource "google_compute_backend_bucket" "exposure_keys_serving" {
  bucket_name = var.bucket_name
  name        = "${var.name_prefix}-backend-bucket"
  enable_cdn  = true
}

resource "google_compute_managed_ssl_certificate" "main" {
  provider = google-beta

  name = "${var.name_prefix}-tls-cert"

  lifecycle {
    create_before_destroy = true
  }

  managed {
    domains = var.domains
  }
}
