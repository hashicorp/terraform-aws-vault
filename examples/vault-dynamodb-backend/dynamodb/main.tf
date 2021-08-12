terraform {
  # This module is now only being tested with Terraform 1.0.x. However, to make upgrading easier, we are setting
  # 0.12.26 as the minimum version, as that version added support for required_providers with source URLs, making it
  # forwards compatible with 1.0.x code.
  required_version = ">= 0.12.26"
}

resource "aws_dynamodb_table" "vault_dynamo" {
  name           = var.table_name
  hash_key       = "Path"
  range_key      = "Key"
  read_capacity  = var.read_capacity
  write_capacity = var.write_capacity

  attribute {
    name = "Path"
    type = "S"
  }

  attribute {
    name = "Key"
    type = "S"
  }
}
