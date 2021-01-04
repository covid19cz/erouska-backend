resource "google_compute_global_address" "main" {
  name        = "${var.name_prefix}-ip"
  description = join(",", var.domains)
}

resource "google_compute_global_forwarding_rule" "http" {
  name       = "${var.name_prefix}-http"
  target     = google_compute_target_http_proxy.http.self_link
  ip_address = google_compute_global_address.main.address
  port_range = "80"
}

resource "google_compute_global_forwarding_rule" "https" {
  name       = "${var.name_prefix}-https"
  target     = google_compute_target_https_proxy.https.self_link
  ip_address = google_compute_global_address.main.address
  port_range = "443"
}

resource "google_compute_target_http_proxy" "http" {
  name    = "${var.name_prefix}-target-proxy"
  url_map = var.https_redirect == false ? google_compute_url_map.main.id : join("", google_compute_url_map.https_redirect.*.self_link)
}

resource "google_compute_target_https_proxy" "https" {
  name    = "${var.name_prefix}-target-proxy-2"
  url_map = google_compute_url_map.main.id

  ssl_certificates = [google_compute_managed_ssl_certificate.main.self_link]
}

resource "google_compute_url_map" "main" {
  name            = var.name_prefix
  default_service = google_compute_backend_service.main.self_link
}

resource "google_compute_url_map" "https_redirect" {
  count = var.https_redirect ? 1 : 0
  name  = "${var.name_prefix}-https-redirect"
  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_backend_service" "main" {
  name                            = "${var.name_prefix}-backend-service"
  enable_cdn                      = true
  connection_draining_timeout_sec = 0

  backend {
    capacity_scaler = 0
    group           = google_compute_region_network_endpoint_group.function_neg.self_link
  }
}

resource "google_compute_region_network_endpoint_group" "function_neg" {
  name                  = "${var.name_prefix}-sneg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_function {
    function = var.function_name
  }
}

resource "google_compute_managed_ssl_certificate" "main" {
  provider = google-beta

  name = replace(var.domains[0], ".", "-")

  lifecycle {
    create_before_destroy = true
  }

  managed {
    domains = var.domains
  }
}
