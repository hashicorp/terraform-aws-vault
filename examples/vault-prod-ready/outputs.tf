output "asg_name_vault_cluster" {
  value = module.vault_cluster.asg_name
}

output "launch_config_name_vault_cluster" {
  value = module.vault_cluster.launch_config_name
}

output "iam_role_arn_vault_cluster" {
  value = module.vault_cluster.iam_role_arn
}

output "iam_role_id_vault_cluster" {
  value = module.vault_cluster.iam_role_id
}

output "security_group_id_vault_cluster" {
  value = module.vault_cluster.security_group_id
}

output "asg_name_consul_cluster" {
  value = module.consul_cluster.asg_name
}

output "launch_config_name_consul_cluster" {
  value = module.consul_cluster.launch_config_name
}

output "iam_role_arn_consul_cluster" {
  value = module.consul_cluster.iam_role_arn
}

output "iam_role_id_consul_cluster" {
  value = module.consul_cluster.iam_role_id
}

output "security_group_id_consul_cluster" {
  value = module.consul_cluster.security_group_id
}

output "aws_region" {
  value = var.aws_region
}

output "vault_servers_cluster_tag_key" {
  value = module.vault_cluster.cluster_tag_key
}

output "vault_servers_cluster_tag_value" {
  value = module.vault_cluster.cluster_tag_value
}

output "ssh_key_name" {
  value = var.ssh_key_name
}

output "vault_cluster_size" {
  value = var.vault_cluster_size
}

output "launch_config_name_servers" {
  value = module.consul_cluster.launch_config_name
}

output "iam_role_arn_servers" {
  value = module.consul_cluster.iam_role_arn
}

output "iam_role_id_servers" {
  value = module.consul_cluster.iam_role_id
}

output "security_group_id_servers" {
  value = module.consul_cluster.security_group_id
}

output "consul_cluster_cluster_tag_key" {
  value = module.consul_cluster.cluster_tag_key
}

output "consul_cluster_cluster_tag_value" {
  value = module.consul_cluster.cluster_tag_value
}

#DNS and Load balancers
output "consul_elb_dns" {
  value = aws_elb.load_balancer_consul.dns_name
}

output "vault_elb_dns" {
  value = aws_elb.load_balancer_vault.dns_name
}
