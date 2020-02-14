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
