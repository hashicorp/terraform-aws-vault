# ---------------------------------------------------------------------------------------------------------------------
# GET CURRENT ACCOUNT INFORMATION
# -------------------------------

data "aws_caller_identity" "current" {}

# ---------------------------------------------------------------------------------------------------------------------
# DEPLOY THE VAULT KMS KEY
# -------------------------

resource "aws_kms_key" "vault_kms_mr_key" {
  deletion_window_in_days  = 30
  description              = "AWS Vault Master Key"
  key_usage                = "ENCRYPT_DECRYPT"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  multi_region             = true
  policy                   = <<POLICY
{
    "Version": "2012-10-17",
    "Id": "vault-key",
    "Statement": [
        {
            "Sid": "Enable IAM User Permissions for owner account",
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
            },
            "Action": "kms:*",
            "Resource": "*"
        },
        {
            "Sid": "Enable IAM User Permissions",
            "Effect": "Allow",
            "Principal": {
                "AWS": "*"
            },
            "Action": [
                "kms:GetPublicKey",
                "kms:Decrypt",
                "kms:ListKeyPolicies",
                "kms:GenerateRandom",
                "kms:ListRetirableGrants",
                "kms:GetKeyPolicy",
                "kms:Verify",
                "kms:ListResourceTags",
                "kms:GenerateDataKeyPair",
                "kms:ReEncryptFrom",
                "kms:ListGrants",
                "kms:GetParametersForImport",
                "kms:DescribeCustomKeyStores",
                "kms:ListKeys",
                "kms:Encrypt",
                "kms:GetKeyRotationStatus",
                "kms:ListAliases",
                "kms:GenerateDataKey",
                "kms:ReEncryptTo",
                "kms:DescribeKey",
                "kms:Sign",
                "kms:CreateGrant"
            ],
            "Resource": "*"
        }
    ]
}
POLICY
}

resource "aws_kms_alias" "vault_kms_mr_key_alias" {
  name          = "alias/multi-region-vault-master-key"
  target_key_id = aws_kms_key.vault_kms_mr_key.key_id
}