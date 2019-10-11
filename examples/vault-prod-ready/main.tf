# ---------------------------------------------------------------------------------------------------------------------
# CONFIGURE OUR AWS CONNECTION
# ---------------------------------------------------------------------------------------------------------------------

provider "aws" {
  # The AWS region in which all resources will be created
  region = var.aws_region

  # Provider version 2.X series is the latest, but has breaking changes with 1.X series.
  version = "~> 2.29"

  # Only these AWS Account IDs may be operated on by this template
  allowed_account_ids = [var.aws_account_id]
}

# ---------------------------------------------------------------------------------------------------------------------
# CONFIGURE REMOTE STATE STORAGE
# ---------------------------------------------------------------------------------------------------------------------

terraform {
  # The configuration for this backend will be filled in by Terragrunt
  backend "s3" {}

  # Only allow this Terraform version. Note that if you upgrade to a newer version, Terraform won't allow you to use an
  # older version, so when you upgrade, you should upgrade everyone on your team and your CI servers all at once.
  required_version = "= 0.12.4"
}
# ----------------------------------------------------------------------------------------------------------------------
# EXTRA
# ----------------------------------------------------------------------------------------------------------------------
# Only allow vault to be accessed from the OpenVPN server
resource "aws_security_group_rule" "allow_vault_inbound_ssh" {
  type                     = "ingress"
  from_port                = 22
  to_port                  = 22
  protocol                 = "tcp"
  source_security_group_id = var.security_group_id
  security_group_id        = module.vault_cluster.security_group_id
}

# Only allow consul to be accessed from the OpenVPN server
resource "aws_security_group_rule" "allow_consul_inbound_ssh" {
  type                     = "ingress"
  from_port                = 22
  to_port                  = 22
  protocol                 = "tcp"
  source_security_group_id = var.security_group_id
  security_group_id        = module.consul_cluster.security_group_id
}

# Only allow vault to be accessed from the OpenVPN server
resource "aws_security_group_rule" "allow_vault_inbound_http" {
  type                     = "ingress"
  from_port                = 8200
  to_port                  = 8200
  protocol                 = "tcp"
  source_security_group_id = var.security_group_id
  security_group_id        = module.vault_cluster.security_group_id
}
# Only allow vault to be accessed from the OpenVPN server
resource "aws_security_group_rule" "allow_consul_inbound_http" {
  type                     = "ingress"
  from_port                = 8500
  to_port                  = 8500
  protocol                 = "tcp"
  source_security_group_id = var.security_group_id
  security_group_id        = module.consul_cluster.security_group_id
}
resource "aws_security_group_rule" "allow_vault_inbound_http_from_elb" {
  type                     = "ingress"
  from_port                = 8200
  to_port                  = 8200
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.elb_vault.id
  security_group_id        = module.vault_cluster.security_group_id
}
resource "aws_security_group_rule" "allow_consul_inbound_http_from_elb" {
  type                     = "ingress"
  from_port                = 8500
  to_port                  = 8500
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.elb_consul.id
  security_group_id        = module.consul_cluster.security_group_id
}

data "aws_kms_alias" "vault-example" {
  name = "alias/${var.auto_unseal_kms_key_alias}"
}

# ---------------------------------------------------------------------------------------------------------------------
# DEPLOY THE VAULT SERVER CLUSTER
# ---------------------------------------------------------------------------------------------------------------------

module "vault_cluster" {
  # When using these modules in your own templates, you will need to use a Git URL with a ref attribute that pins you
  # to a specific version of the modules, such as the following example:
  source = "github.com/hashicorp/terraform-aws-vault.git//modules/vault-cluster?ref=v0.13.3"

  cluster_name  = var.vault_cluster_name
  cluster_size  = var.vault_cluster_size
  instance_type = var.vault_instance_type

  ami_id    = var.ami_id
  user_data = data.template_file.user_data_vault_cluster.rendered

  vpc_id     = var.vpc_id
  subnet_ids = var.vpc_subnet_ids

  # This setting will create the AWS policy that allows the vault cluster to
  # access KMS and use this key for encryption and decryption
  enable_auto_unseal = true

  auto_unseal_kms_key_arn = data.aws_kms_alias.vault-example.target_key_arn

  # To make testing easier, we allow requests from any IP address here but in a production deployment, we *strongly*
  # recommend you limit this to the IP address ranges of known, trusted servers inside your VPC.

  allowed_ssh_cidr_blocks              = []
  allowed_inbound_cidr_blocks          = []
  allowed_inbound_security_group_ids   = []
  allowed_inbound_security_group_count = 0
  ssh_key_name                         = var.ssh_key_name
}

# ---------------------------------------------------------------------------------------------------------------------
# ATTACH IAM POLICIES FOR CONSUL
# To allow our Vault servers to automatically discover the Consul servers, we need to give them the IAM permissions from
# the Consul AWS Module's consul-iam-policies module.
# ---------------------------------------------------------------------------------------------------------------------

module "consul_iam_policies_servers" {
  source = "github.com/hashicorp/terraform-aws-consul.git//modules/consul-iam-policies?ref=v0.7.0"

  iam_role_id = module.vault_cluster.iam_role_id
}

# ---------------------------------------------------------------------------------------------------------------------
# GIVE SSH-GRUNT PERMISSIONS TO TALK TO IAM
# We add an IAM policy to Jenkins that allows ssh-grunt to make API calls to IAM to fetch IAM user and group data.
# ---------------------------------------------------------------------------------------------------------------------

module "ssh_grunt_policies" {
  source = "git::git@github.com:gruntwork-io/module-security.git//modules/iam-policies?ref=v0.19.1"

  aws_account_id = var.aws_account_id

  # ssh-grunt is an automated app, so we can't use MFA with it
  iam_policy_should_require_mfa   = false
  trust_policy_should_require_mfa = false

  # Since our IAM users are defined in a separate AWS account, we need to give ssh-grunt permission to make API calls to
  # that account.
  allow_access_to_other_account_arns = [var.external_account_ssh_grunt_role_arn]
}

resource "aws_iam_role_policy" "ssh_grunt_permissions_vault" {
  name   = "ssh-grunt-permissions"
  role   = module.vault_cluster.iam_role_id
  policy = module.ssh_grunt_policies.allow_access_to_other_accounts[0]
}

resource "aws_iam_role_policy" "ssh_grunt_permissions_consul" {
  name   = "ssh-grunt-permissions"
  role   = module.consul_cluster.iam_role_id
  policy = module.ssh_grunt_policies.allow_access_to_other_accounts[0]
}

# ---------------------------------------------------------------------------------------------------------------------
# ADD IAM POLICY THAT ALLOWS CLOUDWATCH LOG AGGREGATION
# ---------------------------------------------------------------------------------------------------------------------

module "cloudwatch_log_aggregation" {
  source      = "git::git@github.com:gruntwork-io/module-aws-monitoring.git//modules/logs/cloudwatch-log-aggregation-iam-policy?ref=v0.14.0"
  name_prefix = "cloudwatch_consul_vault-${var.vault_cluster_name}"
}

resource "aws_iam_policy_attachment" "attach_cloudwatch_consul_log_aggregation_policy" {
  name       = "attach-cloudwatch-log-aggregation-policy"
  roles      = [module.consul_cluster.iam_role_id]
  policy_arn = module.cloudwatch_log_aggregation.cloudwatch_log_aggregation_policy_arn
}
resource "aws_iam_policy_attachment" "attach_cloudwatch_vault_log_aggregation_policy" {
  name       = "attach-cloudwatch-log-aggregation-policy"
  roles      = [module.vault_cluster.iam_role_id]
  policy_arn = module.cloudwatch_log_aggregation.cloudwatch_log_aggregation_policy_arn
}
# ---------------------------------------------------------------------------------------------------------------------
# ADD IAM POLICY THAT ALLOWS READING AND WRITING CLOUDWATCH METRICS
# ---------------------------------------------------------------------------------------------------------------------

module "cloudwatch_metrics" {
  source      = "git::git@github.com:gruntwork-io/module-aws-monitoring.git//modules/metrics/cloudwatch-custom-metrics-iam-policy?ref=v0.14.0"
  name_prefix = "cloudwatch_consul_vault-${var.consul_cluster_name}"
}

resource "aws_iam_policy_attachment" "attach_cloudwatch_consul_metrics_policy" {
  name       = "attach-cloudwatch-log-aggregation-policy"
  roles      = [module.consul_cluster.iam_role_id]
  policy_arn = module.cloudwatch_log_aggregation.cloudwatch_log_aggregation_policy_arn
}
resource "aws_iam_policy_attachment" "attach_cloudwatch_vault_metrics_policy" {
  name       = "attach-cloudwatch-log-aggregation-policy"
  roles      = [module.vault_cluster.iam_role_id]
  policy_arn = module.cloudwatch_log_aggregation.cloudwatch_log_aggregation_policy_arn
}

# ---------------------------------------------------------------------------------------------------------------------
# THE USER DATA SCRIPT THAT WILL RUN ON EACH VAULT SERVER WHEN IT'S BOOTING
# This script will configure and start Vault
# ---------------------------------------------------------------------------------------------------------------------

data "template_file" "user_data_vault_cluster" {
  template = file("${path.module}/user-data-vault.sh")

  vars = {
    consul_cluster_tag_key   = var.consul_cluster_tag_key
    consul_cluster_tag_value = var.consul_cluster_name
    kms_key_id               = data.aws_kms_alias.vault-example.target_key_id
    aws_region               = var.aws_region
    enable_gossip_encryption = var.enable_gossip_encryption
    gossip_encryption_key    = aws_secretsmanager_secret_version.gossip_encryption_key.secret_string
    enable_rpc_encryption    = var.enable_rpc_encryption
    ca_path                  = var.ca_path
    cert_file_path           = var.cert_file_path
    key_file_path            = var.key_file_path
    consul_token_secret      = aws_secretsmanager_secret.consul_token.name
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# PERMIT CONSUL SPECIFIC TRAFFIC IN VAULT CLUSTER
# To allow our Vault servers consul agents to communicate with other consul agents and participate in the LAN gossip,
# we open up the consul specific protocols and ports for consul traffic
# ---------------------------------------------------------------------------------------------------------------------

module "security_group_rules" {
  source = "github.com/hashicorp/terraform-aws-consul.git//modules/consul-client-security-group-rules?ref=v0.7.0"

  security_group_id = module.vault_cluster.security_group_id

  # To make testing easier, we allow requests from any IP address here but in a production deployment, we *strongly*
  # recommend you limit this to the IP address ranges of known, trusted servers inside your VPC.

  allowed_inbound_cidr_blocks        = []
  allowed_inbound_security_group_ids = []
}

# ---------------------------------------------------------------------------------------------------------------------
# DEPLOY THE CONSUL SERVER CLUSTER
# ---------------------------------------------------------------------------------------------------------------------

module "consul_cluster" {
  source = "github.com/hashicorp/terraform-aws-consul.git//modules/consul-cluster?ref=v0.7.0"

  cluster_name  = var.consul_cluster_name
  cluster_size  = var.consul_cluster_size
  instance_type = var.consul_instance_type

  # The EC2 Instances will use these tags to automatically discover each other and form a cluster
  cluster_tag_key   = var.consul_cluster_tag_key
  cluster_tag_value = var.consul_cluster_name

  ami_id    = var.ami_id
  user_data = data.template_file.user_data_consul.rendered

  vpc_id     = var.vpc_id
  subnet_ids = var.vpc_subnet_ids

  # To make testing easier, we allow Consul and SSH requests from any IP address here but in a production
  # deployment, we strongly recommend you limit this to the IP address ranges of known, trusted servers inside your VPC.

  allowed_ssh_cidr_blocks     = []
  allowed_inbound_cidr_blocks = var.vpc_cidr_blocks
# https://github.com/hashicorp/terraform-aws-vault/pull/115/files allow vault servers by default #115
  allowed_inbound_security_group_count = 1
  allowed_inbound_security_group_ids   = [module.vault_cluster.security_group_id]
  ssh_key_name                = var.ssh_key_name
}

# ---------------------------------------------------------------------------------------------------------------------
# THE USER DATA SCRIPT THAT WILL RUN ON EACH CONSUL SERVER WHEN IT'S BOOTING
# This script will configure and start Consul
# ---------------------------------------------------------------------------------------------------------------------

data "template_file" "user_data_consul" {
  template = file("${path.module}/user-data-consul.sh")

  vars = {
    consul_cluster_tag_key   = var.consul_cluster_tag_key
    consul_cluster_tag_value = var.consul_cluster_name
    aws_region               = var.aws_region
    ca_path                  = var.ca_path
    cert_file_path           = var.cert_file_path
    key_file_path            = var.key_file_path
    enable_gossip_encryption = var.enable_gossip_encryption
    gossip_encryption_key    = aws_secretsmanager_secret_version.gossip_encryption_key.secret_string
    enable_rpc_encryption    = var.enable_rpc_encryption
    consul_token_secret      = aws_secretsmanager_secret.consul_token.name
  }
}

#Add load balancers and dns
# ---------------------------------------------------------------------------------------------------------------------
# CREATE AN ELB TO PERFORM HEALTH CHECKS ON CONSUL
# Use an ELB for health checks. This is useful for doing zero-downtime deployments and making sure that failed nodes
# are automatically replaced. We also use it to expose the management UI.
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_elb" "load_balancer_consul" {
  name            = var.consul_cluster_name
  subnets         = var.vpc_subnet_ids
  security_groups = [aws_security_group.elb_consul.id]
  internal        = true

  connection_draining         = true
  connection_draining_timeout = 60

  # Perform TCP health checks on Consul's client port.
  health_check {
    target              = "TCP:8500"
    interval            = 30
    timeout             = 10
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  # The ELB can be used to reach the management interface
  listener {
    instance_port     = 8500
    instance_protocol = "http"
    lb_port           = 8500
    lb_protocol       = "http"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE A SECURITY GROUP THAT CONTROLS WHAT TRAFFIC CAN GO IN AND OUT OF THE ELB OF CONSUL
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_security_group" "elb_consul" {
  name   = var.consul_cluster_name
  vpc_id = var.vpc_id
}

resource "aws_security_group_rule" "allow_all_outbound_consul" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.elb_consul.id
}

resource "aws_security_group_rule" "allow_consul_inbound_from_elb" {
  type = "ingress"
  from_port = 8500
  to_port = 8500
  protocol = "tcp"
  security_group_id = aws_security_group.elb_consul.id
  source_security_group_id = module.consul_cluster.security_group_id
}

resource "aws_security_group_rule" "allow_openvpntoconsul_inbound_from_elb" {
  type = "ingress"
  from_port = 8500
  to_port = 8500
  protocol = "tcp"
  security_group_id = aws_security_group.elb_consul.id
  source_security_group_id = var.security_group_id
}
## Allow clients to connect from within the provided security group
resource "aws_security_group_rule" "allow_consul_inbound_from_subnets" {
  type = "ingress"
  from_port = 8500
  to_port = 8500
  protocol = "tcp"
  security_group_id = aws_security_group.elb_consul.id
  cidr_blocks = var.vpc_cidr_blocks
}
#Assign ASG to ELB
resource "aws_autoscaling_attachment" "elb_consul" {
  autoscaling_group_name = module.consul_cluster.asg_name
  elb                    = var.consul_cluster_name
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE AN ELB TO PERFORM HEALTH CHECKS ON VAULT
# Use an ELB for health checks. This is useful for doing zero-downtime deployments and making sure that failed nodes
# are automatically replaced. We also use it to expose the management UI.
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_elb" "load_balancer_vault" {
  name                        = var.vault_cluster_name
  subnets                     = var.vpc_subnet_ids
  security_groups             = [aws_security_group.elb_vault.id]
  internal                    = true
  connection_draining         = true
  connection_draining_timeout = 60

  # Perform TCP health checks on Consul's client port.
  health_check {
    target              = "TCP:8200"
    interval            = 30
    timeout             = 10
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  # The ELB can be used to reach the management interface
  listener {
    instance_port     = 8200
    instance_protocol = "http"
    lb_port           = 8200
    lb_protocol       = "http"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE A SECURITY GROUP THAT CONTROLS WHAT TRAFFIC CAN GO IN AND OUT OF THE ELB OF VAULT
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_security_group" "elb_vault" {
  name   = var.vault_cluster_name
  vpc_id = var.vpc_id
}

resource "aws_security_group_rule" "allow_all_outbound_vault" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.elb_vault.id
}

resource "aws_security_group_rule" "allow_vault_inbound_from_elb" {
  type = "ingress"
  from_port = 8200
  to_port = 8200
  protocol = "tcp"
  security_group_id = aws_security_group.elb_vault.id
  source_security_group_id = module.vault_cluster.security_group_id
}

resource "aws_security_group_rule" "allow_openvpntovault_inbound_from_elb" {
  type = "ingress"
  from_port = 8200
  to_port = 8200
  protocol = "tcp"
  security_group_id = aws_security_group.elb_vault.id
  source_security_group_id = var.security_group_id
}
## Allow clients to connect from within the provided security group
resource "aws_security_group_rule" "allow_vault_inbound_from_subnets" {
  type = "ingress"
  from_port = 8200
  to_port = 8200
  protocol = "tcp"
  security_group_id = aws_security_group.elb_vault.id
  cidr_blocks = var.vpc_cidr_blocks
}
#Assign ASG to ELB
resource "aws_autoscaling_attachment" "elb_vault" {
  autoscaling_group_name = module.vault_cluster.asg_name
  elb                    = var.vault_cluster_name
}

# Secret that holds the Consul master token
resource "aws_secretsmanager_secret" "consul_token" {
  name_prefix = "${var.consul_cluster_name}-token"
}

# Random uuid used as master token
resource "random_uuid" "consul_token" {}

# Secret version updated with the random uuid
resource "aws_secretsmanager_secret_version" "consul_token" {
  secret_id     = aws_secretsmanager_secret.consul_token.id
  secret_string = random_uuid.consul_token.result
}

# Secret that holds the gossip encryption key
resource "aws_secretsmanager_secret" "gossip_encryption_key" {
  name_prefix = "gossip_encryption_key"
}

# Random uuid used as gossip encryption key
resource "random_string" "gossip_encryption_key" {
  length = 32

}

# Secret version updated with the random uuid
resource "aws_secretsmanager_secret_version" "gossip_encryption_key" {
  secret_id     = aws_secretsmanager_secret.gossip_encryption_key.id
  secret_string = base64encode(random_string.gossip_encryption_key.result)
}

# Policy to allow Consul to write the consul_token secret and gossip encryption key
resource "aws_iam_policy" "secretsmanager_get_token" {
  name   = var.consul_cluster_name
  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue"
            ],
            "Resource": [
                "${aws_secretsmanager_secret.consul_token.arn}",
                "${aws_secretsmanager_secret.gossip_encryption_key.arn}"
            ]
        }
    ]
}
EOF
}

# Attach the policy to the roles of the Consul instances
resource "aws_iam_role_policy_attachment" "consul_secretsmanager" {
  role = module.consul_cluster.iam_role_id
  policy_arn = aws_iam_policy.secretsmanager_get_token.arn
}

# Attach the policy to the roles of the Vault instances
resource "aws_iam_role_policy_attachment" "vault_secretsmanager" {
  role = module.vault_cluster.iam_role_id
  policy_arn = aws_iam_policy.secretsmanager_get_token.arn
}

data "aws_caller_identity" "current" {}
