module "gce-container" {
  source = "github.com/terraform-google-modules/terraform-google-container-vm"
  #version = "0.1.0"

  container = {
    image = "gcr.io/${var.project}/${var.image}"
    args = [
      "server",
      "--atlantis-url=https://${google_compute_address.static.address}",
      "--gh-user=${var.github_user}",
      "--gh-token=${var.github_token}",
      "--gh-webhook-secret=${var.webhook_secret}",
      "--repo-allowlist=${var.repo_allowlist}",
      "--ssl-cert-file=/etc/atlantis/tls/tls.cert",
      "--ssl-key-file=/etc/atlantis/tls/tls.key",
    ]
  }

  restart_policy = "Always"
}

resource "null_resource" "build" {
  depends_on = [tls_private_key.key, tls_locally_signed_cert.cert]
  provisioner "local-exec" {
    command = "docker build  --build-arg ATLANTIS_IMAGE=${var.image} -t gcr.io/${var.project}/${var.image} . && docker push gcr.io/${var.project}/${var.image}"
  }
}
