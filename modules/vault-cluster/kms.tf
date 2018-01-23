data "aws_kms_alias" "vault" {
    name = "${var.kms_key_alias}"
}
