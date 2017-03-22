# ---------------------------------------------------------------------------------------------------------------------
# CREATE A SELF-SIGNED TLS CERTIFICATE
# ---------------------------------------------------------------------------------------------------------------------

resource "tls_private_key" "cert" {
  algorithm   = "${var.private_key_algorithm}"
  ecdsa_curve = "${var.private_key_ecdsa_curve}"

  # Store the private key in a file.
  provisioner "local-exec" {
    command = "echo '${tls_private_key.cert.private_key_pem}' > '${var.private_key_file_path}' && chmod ${var.permissions} '${var.private_key_file_path}' && chown ${var.owner} '${var.private_key_file_path}'"
  }
}

resource "tls_self_signed_cert" "cert" {
  key_algorithm     = "${tls_private_key.cert.algorithm}"
  private_key_pem   = "${tls_private_key.cert.private_key_pem}"
  is_ca_certificate = false

  validity_period_hours = "${var.validity_period_hours}"
  allowed_uses          = ["${var.allowed_uses}"]

  dns_names    = ["${var.dns_names}"]
  ip_addresses = ["${var.ip_addresses}"]

  subject {
    common_name  = "${var.common_name}"
    organization = "${var.organization_name}"
  }

  # Store the public key in a file.
  provisioner "local-exec" {
    command = "echo '${tls_self_signed_cert.cert.cert_pem}' > '${var.public_key_file_path}' && chmod ${var.permissions} '${var.public_key_file_path}' && chown ${var.owner} '${var.public_key_file_path}'"
  }
}

