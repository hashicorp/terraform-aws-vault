output "vault_fully_qualified_domain_name" {
  value = module.vault_elb.fully_qualified_domain_name
}

output "vault_elb_dns_name" {
  value = module.vault_elb.load_balancer_dns_name
}

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
  value = data.aws_region.current.name
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

