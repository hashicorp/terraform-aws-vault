# ---------------------------------------------------------------------------------------------------------------------
# REQUIRED PARAMETERS
# You must provide a value for each of these parameters.
# ---------------------------------------------------------------------------------------------------------------------

variable "name" {
  description = "The name to use for the ELB and all other resources in this module."
}

variable "vpc_id" {
  description = "The ID of the VPC into which the ELB should be deployed."
}

variable "allowed_inbound_cidr_blocks" {
  description = "A list of CIDR-formatted IP address ranges from which the ELB will accept requests."
  type        = "list"
}

variable "vault_asg_name" {
  description = "Name of the Vault Autoscaling Group"
}

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONAL PARAMETERS
# These parameters have reasonable defaults.
# ---------------------------------------------------------------------------------------------------------------------

variable "subnet_ids" {
  description = "The subnet IDs into which the ELB should be deployed. You will typically want to deploy the ELB into public subnets so your Vault cluster can run in private subnets. At least one of var.subnet_ids or var.availability_zones must be non-empty."
  type        = "list"
  default     = []
}

variable "create_dns_entry" {
  description = "If set to true, this module will create a Route 53 DNS A record for the ELB in the var.hosted_zone_id hosted zone with the domain name in var.domain_name."
  default     = false
}

variable "hosted_zone_id" {
  description = "The Route 53 hosted zone ID in which to create a DNS A record. Only used if var.create_dns_entry is true."
  default     = "replace-me"
}

variable "domain_name" {
  description = "The domain name to use in the DNS A record for the ELB (e.g. vault.example.com). Only used if var.create_dns_entry is true."
  default     = "replace-me"
}

variable "lb_port" {
  description = "The port the load balancer should listen on for API requests."
  default     = 443
}

variable "vault_api_port" {
  description = "The port to listen on for API requests."
  default     = 8200
}

variable "internal" {
  description = "If set to true, this will be an internal ELB, accessible only within the VPC. The main reason to use an ELB with Vault is to make it publicly accessible, so this should typically be set to false."
  default     = false
}

variable "cross_zone_load_balancing" {
  description = "Set to true to enable cross-zone load balancing"
  default     = true
}

variable "idle_timeout" {
  description = "The time, in seconds, that the connection is allowed to be idle."
  default     = 60
}

variable "connection_draining" {
  description = "Set to true to enable connection draining."
  default     = true
}

variable "connection_draining_timeout" {
  description = "The time, in seconds, to allow for connections to drain."
  default     = 300
}

variable "health_check_protocol" {
  description = "The protocol to use for health checks. Must be one of: HTTP, HTTPS, TCP, SSL."
  default     = "HTTPS"
}

variable "health_check_path" {
  description = "The path to use for health checks. Must return a 200 OK when the service is ready to receive requests from the ELB."
  default     = "/v1/sys/health?standbyok=true"
}

variable "health_check_port" {
  description = "The port to use for health checks if not vault_api_port."
  default     = 0
}

variable "health_check_interval" {
  description = "The amount of time, in seconds, between health checks."
  default     = 15
}

variable "health_check_healthy_threshold" {
  description = "The number of health checks that must pass before the instance is declared healthy."
  default     = 2
}

variable "health_check_unhealthy_threshold" {
  description = "The number of health checks that must fail before the instance is declared unhealthy."
  default     = 2
}

variable "health_check_timeout" {
  description = "The amount of time, in seconds, before a health check times out."
  default     = 5
}

variable "load_balancer_tags" {
  description = "Tags to be applied to the ELB."
  default     = {}
}

variable "security_group_tags" {
  description = "Tags to be applied to the ELB security group."
  type        = "map"
  default     = {}
}
