# ---------------------------------------------------------------------------------------------------------------------
# ENVIRONMENT VARIABLES
# Define these secrets as environment variables
# ---------------------------------------------------------------------------------------------------------------------

# AWS_ACCESS_KEY_ID
# AWS_SECRET_ACCESS_KEY
# AWS_DEFAULT_REGION

# ---------------------------------------------------------------------------------------------------------------------
# REQUIRED PARAMETERS
# You must provide a value for each of these parameters.
# ---------------------------------------------------------------------------------------------------------------------

variable "ami_id" {
  description = "The ID of the AMI to run in the cluster. This should be an AMI built from the Packer template under examples/vault-consul-ami/vault-consul.json."
}

variable "ssh_key_name" {
  description = "The name of an EC2 Key Pair that can be used to SSH to the EC2 Instances in this cluster. Set to an empty string to not associate a Key Pair."
}

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONAL PARAMETERS
# These parameters have reasonable defaults.
# ---------------------------------------------------------------------------------------------------------------------

variable "vault_cluster_name" {
  description = "What to name the Vault server cluster and all of its associated resources"
  default     = "vault-s3-example"
}

variable "consul_cluster_name" {
  description = "What to name the Consul server cluster and all of its associated resources"
  default     = "consul-s3-example"
}

variable "vault_cluster_size" {
  description = "The number of Vault server nodes to deploy. We strongly recommend using 3 or 5."
  default     = 3
}

variable "consul_cluster_size" {
  description = "The number of Consul server nodes to deploy. We strongly recommend using 3 or 5."
  default     = 3
}

variable "vault_instance_type" {
  description = "The type of EC2 Instance to run in the Vault ASG"
  default     = "t2.micro"
}

variable "consul_instance_type" {
  description = "The type of EC2 Instance to run in the Consul ASG"
  default     = "t2.micro"
}

variable "consul_cluster_tag_key" {
  description = "The tag the Consul EC2 Instances will look for to automatically discover each other and form a cluster."
  default     = "consul-vault-s3-servers"
}

variable "vpc_id" {
  description = "The ID of the VPC to deploy into. Leave an empty string to use the Default VPC in this region."
  default     = ""
}

variable "s3_bucket_name" {
  description = "The name of an S3 bucket to create and use as a storage backend (if configured). Note: S3 bucket names must be *globally* unique."
  default     = "my-vault-bucket"
}

variable "force_destroy_s3_bucket" {
  description = "If you set this to true, when you run terraform destroy, this tells Terraform to delete all the objects in the S3 bucket used for backend storage (if configured). You should NOT set this to true in production or you risk losing all your data! This property is only here so automated tests of this module can clean up after themselves."
  default     = false
}
