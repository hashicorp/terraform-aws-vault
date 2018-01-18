resource "aws_kms_key" "vault" {
    description         = "${var.cluster_name} KMS Master Key"
}

resource "aws_kms_alias" "vault" {
    name            = "alias/${var.cluster_name}-vault"
    target_key_id   = "${aws_kms_key.vault.key_id}"
}