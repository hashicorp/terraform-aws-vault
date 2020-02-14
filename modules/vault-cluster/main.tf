# ----------------------------------------------------------------------------------------------------------------------
# REQUIRE A SPECIFIC TERRAFORM VERSION OR HIGHER
# This module has been updated with 0.12 syntax, which means it is no longer compatible with any versions below 0.12.
# ----------------------------------------------------------------------------------------------------------------------
terraform {
  required_version = ">= 0.12"
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE AN AUTO SCALING GROUP (ASG) TO RUN VAULT
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_autoscaling_group" "autoscaling_group" {
  name_prefix = var.cluster_name

  launch_configuration = aws_launch_configuration.launch_configuration.name

  availability_zones  = var.availability_zones
  vpc_zone_identifier = var.subnet_ids

  # Use a fixed-size cluster
  min_size             = var.cluster_size
  max_size             = var.cluster_size
  desired_capacity     = var.cluster_size
  termination_policies = [var.termination_policies]

  health_check_type         = var.health_check_type
  health_check_grace_period = var.health_check_grace_period
  wait_for_capacity_timeout = var.wait_for_capacity_timeout

  enabled_metrics = var.enabled_metrics

  # Use bucket and policies names in tags for depending on them when they are there
  # And only create the cluster after S3 bucket and policies exist
  # Otherwise Vault might boot and not find the bucket or not yet have the necessary permissions
  # Not using `depends_on` because these resources might not exist
  tag {
    key                 = var.cluster_tag_key
    value               = var.cluster_name
    propagate_at_launch = true
  }

  tag {
    key                 = "using_s3_bucket_backend"
    value               = element(concat(aws_iam_role_policy.vault_s3.*.name, [""]), 0)
    propagate_at_launch = true
  }

  tag {
    key                 = "s3_bucket_id"
    value               = element(concat(aws_s3_bucket.vault_storage.*.id, [""]), 0)
    propagate_at_launch = true
  }


  # Use table policy name in tags for depending on them when they are there
  # And only create the cluster after dynamo exists
  # Otherwise Vault might boot and not find the bucket or not yet have the necessary permissions
  # Not using `depends_on` because these resources might not exist
  tag {
    key                 = "using_dynamodb_backend"
    value               = element(concat(aws_iam_role_policy.vault_dynamo.*.name, [""]), 0)
    propagate_at_launch = true
  }

  tag {
    key                 = "using_auto_unseal"
    value               = element(concat(aws_iam_role_policy.vault_auto_unseal_kms.*.name, [""]), 0)
    propagate_at_launch = true
  }

  dynamic "tag" {
    for_each = var.cluster_extra_tags

    content {
      key                 = tag.value.key
      value               = tag.value.value
      propagate_at_launch = tag.value.propagate_at_launch
    }
  }


  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE LAUNCH CONFIGURATION TO DEFINE WHAT RUNS ON EACH INSTANCE IN THE ASG
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_launch_configuration" "launch_configuration" {
  name_prefix   = "${var.cluster_name}-"
  image_id      = var.ami_id
  instance_type = var.instance_type
  user_data     = var.user_data

  iam_instance_profile = aws_iam_instance_profile.instance_profile.name
  key_name             = var.ssh_key_name
  # TF-UPGRADE-TODO: In Terraform v0.10 and earlier, it was sometimes necessary to
  # force an interpolation expression to be interpreted as a list by wrapping it
  # in an extra set of list brackets. That form was supported for compatibilty in
  # v0.11, but is no longer supported in Terraform v0.12.
  #
  # If the expression in the following list itself returns a list, remove the
  # brackets to avoid interpretation as a list of lists. If the expression
  # returns a single list item then leave it as-is and remove this TODO comment.
  security_groups = concat(
    [aws_security_group.lc_security_group.id],
    var.additional_security_group_ids,
  )
  placement_tenancy           = var.tenancy
  associate_public_ip_address = var.associate_public_ip_address

  ebs_optimized = var.root_volume_ebs_optimized

  root_block_device {
    volume_type           = var.root_volume_type
    volume_size           = var.root_volume_size
    delete_on_termination = var.root_volume_delete_on_termination
  }

  # Important note: whenever using a launch configuration with an auto scaling group, you must set
  # create_before_destroy = true. However, as soon as you set create_before_destroy = true in one resource, you must
  # also set it in every resource that it depends on, or you'll get an error about cyclic dependencies (especially when
  # removing resources). For more info, see:
  #
  # https://www.terraform.io/docs/providers/aws/r/launch_configuration.html
  # https://terraform.io/docs/configuration/resources.html
  lifecycle {
    create_before_destroy = true
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE A SECURITY GROUP TO CONTROL WHAT REQUESTS CAN GO IN AND OUT OF EACH EC2 INSTANCE
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_security_group" "lc_security_group" {
  name_prefix = var.cluster_name
  description = "Security group for the ${var.cluster_name} launch configuration"
  vpc_id      = var.vpc_id

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }

  tags = merge(
    {
      "Name" = var.cluster_name
    },
    var.security_group_tags,
  )
}

resource "aws_security_group_rule" "allow_ssh_inbound_from_cidr_blocks" {
  count       = length(var.allowed_ssh_cidr_blocks) >= 1 ? 1 : 0
  type        = "ingress"
  from_port   = var.ssh_port
  to_port     = var.ssh_port
  protocol    = "tcp"
  cidr_blocks = var.allowed_ssh_cidr_blocks

  security_group_id = aws_security_group.lc_security_group.id
}

resource "aws_security_group_rule" "allow_ssh_inbound_from_security_group_ids" {
  count                    = length(var.allowed_ssh_security_group_ids)
  type                     = "ingress"
  from_port                = var.ssh_port
  to_port                  = var.ssh_port
  protocol                 = "tcp"
  source_security_group_id = element(var.allowed_ssh_security_group_ids, count.index)

  security_group_id = aws_security_group.lc_security_group.id
}

resource "aws_security_group_rule" "allow_all_outbound" {
  type        = "egress"
  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.lc_security_group.id
}

# ---------------------------------------------------------------------------------------------------------------------
# THE INBOUND/OUTBOUND RULES FOR THE SECURITY GROUP COME FROM THE VAULT-SECURITY-GROUP-RULES MODULE
# ---------------------------------------------------------------------------------------------------------------------

module "security_group_rules" {
  source = "../vault-security-group-rules"

  security_group_id                    = aws_security_group.lc_security_group.id
  allowed_inbound_cidr_blocks          = var.allowed_inbound_cidr_blocks
  allowed_inbound_security_group_ids   = var.allowed_inbound_security_group_ids
  allowed_inbound_security_group_count = var.allowed_inbound_security_group_count

  api_port     = var.api_port
  cluster_port = var.cluster_port
}

# ---------------------------------------------------------------------------------------------------------------------
# ATTACH AN IAM ROLE TO EACH EC2 INSTANCE
# We can use the IAM role to grant the instance IAM permissions so we can use the AWS APIs without having to figure out
# how to get our secret AWS access keys onto the box.
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_iam_instance_profile" "instance_profile" {
  name_prefix = var.cluster_name
  path        = var.instance_profile_path
  role        = aws_iam_role.instance_role.name

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_iam_role" "instance_role" {
  name_prefix        = var.cluster_name
  assume_role_policy = data.aws_iam_policy_document.instance_role.json

  # aws_iam_instance_profile.instance_profile in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

data "aws_iam_policy_document" "instance_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_s3_bucket" "vault_storage" {
  count         = var.enable_s3_backend ? 1 : 0
  bucket        = var.s3_bucket_name
  force_destroy = var.force_destroy_s3_bucket

  tags = merge(
    {
      "Description" = "Used for secret storage with Vault. DO NOT DELETE this Bucket unless you know what you are doing."
    },
    var.s3_bucket_tags,
  )

  versioning {
    enabled = var.enable_s3_bucket_versioning
  }

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_iam_role_policy" "vault_s3" {
  count = var.enable_s3_backend ? 1 : 0
  name  = "vault_s3"
  role  = aws_iam_role.instance_role.id
  policy = element(
    concat(data.aws_iam_policy_document.vault_s3.*.json, [""]),
    0,
  )

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

data "aws_iam_policy_document" "vault_s3" {
  count = var.enable_s3_backend ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["s3:*"]

    resources = [
      aws_s3_bucket.vault_storage[0].arn,
      "${aws_s3_bucket.vault_storage[0].arn}/*",
    ]
  }
}

data "aws_iam_policy_document" "vault_dynamo" {
  count = var.enable_dynamo_backend ? 1 : 0
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
    resources = [
      format("arn:aws:dynamodb:%s:%s:table/%s",
        var.dynamo_table_region == "" ? data.aws_region.current.name : var.dynamo_table_region,
        data.aws_caller_identity.current.account_id,
        var.dynamo_table_name
      )
    ]
  }
}

resource "aws_iam_role_policy" "vault_dynamo" {
  count  = var.enable_dynamo_backend ? 1 : 0
  name   = "vault_dynamo"
  role   = aws_iam_role.instance_role.id
  policy = element(
    concat(data.aws_iam_policy_document.vault_dynamo.*.json, [""]),
    0,
  )
}

data "aws_iam_policy_document" "vault_auto_unseal_kms" {
  count = var.enable_auto_unseal ? 1 : 0

  statement {
    effect = "Allow"

    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:DescribeKey",
    ]

    resources = [var.auto_unseal_kms_key_arn]
  }
}

resource "aws_iam_role_policy" "vault_auto_unseal_kms" {
  count = var.enable_auto_unseal ? 1 : 0
  name  = "vault_auto_unseal_kms"
  role  = aws_iam_role.instance_role.id
  policy = element(
    concat(
      data.aws_iam_policy_document.vault_auto_unseal_kms.*.json,
      [""],
    ),
    0,
  )

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

