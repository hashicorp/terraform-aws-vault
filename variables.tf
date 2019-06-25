# ---------------------------------------------------------------------------------------------------------------------
# ENVIRONMENT VARIABLES
# Define these secrets as environment variables
# ---------------------------------------------------------------------------------------------------------------------

# AWS_ACCESS_KEY_ID
# AWS_SECRET_ACCESS_KEY
# AWS_DEFAULT_REGION

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONAL PARAMETERS
# These parameters have reasonable defaults.
# ---------------------------------------------------------------------------------------------------------------------

variable "create_dns_entry" {
  description = "If set to true, this module will create a Route 53 DNS A record for the ELB in the var.hosted_zone_id hosted zone with the domain name in var.vault_domain_name."
  type        = bool
  default     = false
}

variable "hosted_zone_domain_name" {
  description = "The domain name of the Route 53 Hosted Zone in which to add a DNS entry for Vault (e.g. example.com). Only used if var.create_dns_entry is true."
  type        = string
  default     = null
}

variable "vault_domain_name" {
  description = "The domain name to use in the DNS A record for the Vault ELB (e.g. vault.example.com). Make sure that a) this is a domain within the var.hosted_zone_domain_name hosted zone and b) this is the same domain name you used in the TLS certificates for Vault. Only used if var.create_dns_entry is true."
  type        = string
  default     = null
}

variable "ami_id" {
  description = "The ID of the AMI to run in the cluster. This should be an AMI built from the Packer template under examples/vault-consul-ami/vault-consul.json. If no AMI is specified, the template will 'just work' by using the example public AMIs. WARNING! Do not use the example AMIs in a production setting!"
  type        = string
  default     = null
}

variable "ssh_key_name" {
  description = "The name of an EC2 Key Pair that can be used to SSH to the EC2 Instances in this cluster. Set to an empty string to not associate a Key Pair."
  type        = string
  default     = null
}

variable "subnet_tags" {
  description = "Tags used to find subnets for vault and consul servers"
  type        = map(string)
  default     = {}
}

variable "vpc_tags" {
  description = "Tags used to find a vpc for building resources in"
  type        = map(string)
  default     = {}
}

variable "use_default_vpc" {
  description = "Whether to use the default VPC - NOT recommended for production! - should more likely change this to false and use the vpc_tags to find your vpc"
  type        = bool
  default     = true
}

variable "vault_cluster_name" {
  description = "What to name the Vault server cluster and all of its associated resources"
  type        = string
  default     = "vault-example"
}

variable "consul_cluster_name" {
  description = "What to name the Consul server cluster and all of its associated resources"
  type        = string
  default     = "consul-example"
}

variable "vault_cluster_size" {
  description = "The number of Vault server nodes to deploy. We strongly recommend using 3 or 5."
  type        = number
  default     = 3
}

variable "consul_cluster_size" {
  description = "The number of Consul server nodes to deploy. We strongly recommend using 3 or 5."
  type        = number
  default     = 3
}

variable "vault_instance_type" {
  description = "The type of EC2 Instance to run in the Vault ASG"
  type        = string
  default     = "t2.micro"
}

variable "consul_instance_type" {
  description = "The type of EC2 Instance to run in the Consul ASG"
  type        = string
  default     = "t2.nano"
}

variable "consul_cluster_tag_key" {
  description = "The tag the Consul EC2 Instances will look for to automatically discover each other and form a cluster."
  type        = string
  default     = "consul-servers"
}

