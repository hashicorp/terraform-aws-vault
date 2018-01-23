data "aws_kms_alias" "vault" {
    name = "alias/${var.kms_key_alias}"
}
