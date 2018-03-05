variable "hosted_zone_id" {
  description = "The Route 53 hosted zone ID in which to create a DNS A records and SRV record."
}

variable "instance_ips" {
  description = "The instance IPs for which to create A records."
  type = "list"
}

variable "domain_names" {
  description = "The domain name to use for an A record to each instance IP."
  type = "list"
}

variable "srv_domain_name" {
  description = "The domain name to use for a SRV record resolving to all instance names."
  default = ""
}

variable "api_port" {
  description = "The port the Vault instances are listening on for API requests."
  default     = 8200
}

variable "ttl" {
  default = 60
}

variable "protocol" {
  description = "The Vault API protocol (http or https, for use in SRV 'service' name)"
  default = "https"
}
