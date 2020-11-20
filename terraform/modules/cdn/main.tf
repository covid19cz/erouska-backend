resource "google_compute_global_address" "main" {
  name = "${var.name_prefix}-ip"
}

resource "google_compute_global_forwarding_rule" "main" {
  name       = "${var.name_prefix}-lb"
  target     = google_compute_target_http_proxy.main.self_link
  ip_address = google_compute_global_address.main.address
  port_range = "80"
}

resource "google_compute_target_http_proxy" "main" {
  name    = "${var.name_prefix}-serving-target-proxy"
  url_map = google_compute_url_map.main.id
}

# resource "google_compute_target_htts_proxy" "main" {
#   name    = "${var.name_prefix}-serving-target-proxy"
#   url_map = google_compute_url_map.main.id

#   ssl_certificates = [google_compute_ssl_certificate.main.self_link]
# }

resource "google_compute_url_map" "main" {
  name            = "${var.name_prefix}-serving"
  default_service = google_compute_backend_bucket.exposure_keys_serving.self_link
}

resource "google_compute_backend_bucket" "exposure_keys_serving" {
  bucket_name = var.bucket_name
  name        = "${var.name_prefix}-backend-bucket"
  enable_cdn  = true
}
