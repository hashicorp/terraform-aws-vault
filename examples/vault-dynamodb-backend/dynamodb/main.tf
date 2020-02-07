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

data "aws_iam_policy_document" "vault_dynamo" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:DescribeLimits",
      "dynamodb:DescribeTimeToLive",
      "dynamodb:ListTagsOfResource",
      "dynamodb:DescribeReservedCapacityOfferings",
      "dynamodb:DescribeReservedCapacity",
      "dynamodb:ListTables",
      "dynamodb:BatchGetItem",
      "dynamodb:BatchWriteItem",
      "dynamodb:CreateTable",
      "dynamodb:DeleteItem",
      "dynamodb:GetItem",
      "dynamodb:GetRecords",
      "dynamodb:PutItem",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
      "dynamodb:Scan",
      "dynamodb:DescribeTable"
    ]
    resources = [aws_dynamodb_table.vault_dynamo.arn]
  }
}