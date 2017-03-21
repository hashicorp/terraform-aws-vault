# ---------------------------------------------------------------------------------------------------------------------
# CREATE A CA CERTIFICATE
# ---------------------------------------------------------------------------------------------------------------------

resource "tls_private_key" "ca" {
  algorithm = "${var.private_key_algorithm}"

  # Store the CA certificate's private key in a file.
  provisioner "local-exec" {
    command = "echo '${tls_private_key.ca.private_key_pem}' > '${var.ca_cert_private_key_file_path}' && chmod ${var.cert_permissions} '${var.ca_cert_private_key_file_path}' && chown ${var.cert_owner} '${var.ca_cert_private_key_file_path}'"
  }
}

resource "tls_self_signed_cert" "ca" {
  key_algorithm     = "${tls_private_key.ca.algorithm}"
  private_key_pem   = "${tls_private_key.ca.private_key_pem}"
  is_ca_certificate = true

  validity_period_hours = "${var.validity_period_hours}"
  allowed_uses          = ["${var.ca_cert_allowed_uses}"]

  subject {
    common_name  = "${var.ca_common_name}"
    organization = "${var.organization_name}"
  }

  # Store the CA certificate's public key in a file.
  provisioner "local-exec" {
    command = "echo '${tls_self_signed_cert.ca.cert_pem}' > '${var.ca_cert_public_key_file_path}' && chmod ${var.cert_permissions} '${var.ca_cert_public_key_file_path}' && chown ${var.cert_owner} '${var.ca_cert_public_key_file_path}'"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE A TLS CERTIFICATE
# ---------------------------------------------------------------------------------------------------------------------

resource "tls_private_key" "cert" {
  algorithm = "${var.private_key_algorithm}"

  # Store the certificate's private key in a file
  provisioner "local-exec" {
    command = "echo '${tls_private_key.cert.private_key_pem}' > '${var.cert_private_key_file_path}' && chmod ${var.cert_permissions} '${var.cert_private_key_file_path}' && chown ${var.cert_owner} '${var.cert_private_key_file_path}'"
  }
}

resource "tls_cert_request" "cert" {
  key_algorithm   = "${tls_private_key.cert.algorithm}"
  private_key_pem = "${tls_private_key.cert.private_key_pem}"

  dns_names    = ["${var.dns_names}"]
  ip_addresses = ["${var.ip_addresses}"]

  subject {
    common_name  = "${var.cert_common_name}"
    organization = "${var.organization_name}"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# SIGN THE TLS CERTIFICATE WITH THE CA CERTIFICATE
# ---------------------------------------------------------------------------------------------------------------------

resource "tls_locally_signed_cert" "cert" {
  cert_request_pem = "${tls_cert_request.cert.cert_request_pem}"

  ca_key_algorithm   = "${tls_private_key.ca.algorithm}"
  ca_private_key_pem = "${tls_private_key.ca.private_key_pem}"
  ca_cert_pem        = "${tls_self_signed_cert.ca.cert_pem}"

  validity_period_hours = "${var.validity_period_hours}"
  allowed_uses          = ["${var.cert_allowed_uses}"]

  # Store the certificates public key in a file
  provisioner "local-exec" {
    command = "echo '${tls_locally_signed_cert.cert.cert_pem}' > '${var.cert_public_key_file_path}' && chmod ${var.cert_permissions} '${var.cert_public_key_file_path}' && chown ${var.cert_owner} '${var.cert_public_key_file_path}'"
  }
}
