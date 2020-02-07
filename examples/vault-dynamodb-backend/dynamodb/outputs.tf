output "backend_policy" {
  description = "Policy for the instance"
  value       = data.aws_iam_policy_document.vault_dynamo.json
}
