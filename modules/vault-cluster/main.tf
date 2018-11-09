# ---------------------------------------------------------------------------------------------------------------------
# THESE TEMPLATES REQUIRE TERRAFORM VERSION 0.8 AND ABOVE
# ---------------------------------------------------------------------------------------------------------------------

terraform {
  required_version = ">= 0.9.3"
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE AN AUTO SCALING GROUP (ASG) TO RUN VAULT
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_autoscaling_group" "autoscaling_group" {
  count       = "${var.asg_launch_mechanism == "launch_configuration" ? 1 : 0}"
  name_prefix = "${var.cluster_name}"

  launch_configuration = "${aws_launch_configuration.launch_configuration.name}"

  depends_on = ["aws_iam_instance_profile.instance_profile", "aws_launch_configuration.launch_configuration"]

  availability_zones  = ["${var.availability_zones}"]
  vpc_zone_identifier = ["${var.subnet_ids}"]

  # Use a fixed-size cluster
  min_size             = "${var.cluster_size}"
  max_size             = "${var.cluster_size}"
  desired_capacity     = "${var.cluster_size}"
  termination_policies = ["${var.termination_policies}"]

  health_check_type         = "${var.health_check_type}"
  health_check_grace_period = "${var.health_check_grace_period}"
  wait_for_capacity_timeout = "${var.wait_for_capacity_timeout}"

  tags = ["${concat(
    var.cluster_extra_tags,
    list(
      map("key", var.cluster_tag_key, "value", var.cluster_name, "propagate_at_launch", true)
      )
    )
  }"]
}

# An alternate autoscaling group that uses a launch_template
resource "aws_autoscaling_group" "lt_autoscaling_group" {
  count       = "${var.asg_launch_mechanism == "launch_template" ? 1 : 0}"
  name_prefix = "${var.cluster_name}"

  launch_template {
    id      = "${aws_launch_template.launch_template.id}"
    version = "${var.launch_template_version}"
  }

  depends_on = ["aws_iam_instance_profile.instance_profile", "aws_launch_template.launch_template"]

  availability_zones  = ["${var.availability_zones}"]
  vpc_zone_identifier = ["${var.subnet_ids}"]

  # Use a fixed-size cluster
  min_size             = "${var.cluster_size}"
  max_size             = "${var.cluster_size}"
  desired_capacity     = "${var.cluster_size}"
  termination_policies = ["${var.termination_policies}"]

  health_check_type         = "${var.health_check_type}"
  health_check_grace_period = "${var.health_check_grace_period}"
  wait_for_capacity_timeout = "${var.wait_for_capacity_timeout}"

  tags = ["${concat(
    var.cluster_extra_tags,
    list(
      map("key", var.cluster_tag_key, "value", var.cluster_name, "propagate_at_launch", true)
      )
    )
  }"]

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
  count         = "${var.asg_launch_mechanism == "launch_configuration" ? 1 : 0}"
  name_prefix   = "${var.cluster_name}-"
  image_id      = "${var.ami_id}"
  instance_type = "${var.instance_type}"
  user_data     = "${var.user_data}"

  iam_instance_profile        = "${aws_iam_instance_profile.instance_profile.name}"
  key_name                    = "${var.ssh_key_name}"
  security_groups             = ["${concat(list(aws_security_group.lc_security_group.id), var.additional_security_group_ids)}"]
  placement_tenancy           = "${var.tenancy}"
  associate_public_ip_address = "${var.associate_public_ip_address}"

  ebs_optimized = "${var.root_volume_ebs_optimized}"

  root_block_device {
    volume_type           = "${var.root_volume_type}"
    volume_size           = "${var.root_volume_size}"
    delete_on_termination = "${var.root_volume_delete_on_termination}"
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
# CREATE LAUNCH TEMPLATE TO DEFINE WHAT RUNS ON EACH INSTANCE IN THE ASG
# ---------------------------------------------------------------------------------------------------------------------

data "aws_ami" "ami" {
  filter {
    name   = "image-id"
    values = ["${var.ami_id}"]
  }
}

resource "aws_launch_template" "launch_template" {
  count         = "${var.asg_launch_mechanism == "launch_template" ? 1 : 0}"
  name_prefix   = "${var.cluster_name}-"
  image_id      = "${var.ami_id}"
  instance_type = "${var.instance_type}"
  user_data     = "${base64encode(var.user_data)}"

  depends_on = ["aws_iam_instance_profile.instance_profile"]

  iam_instance_profile {
    name = "${aws_iam_instance_profile.instance_profile.name}"
  }

  key_name = "${var.ssh_key_name}"

  # Don't use vpc_security_group_ids when network_interfaces includes security_groups.
  # vpc_security_group_ids = ["${concat(list(aws_security_group.lc_security_group.id), var.additional_security_group_ids)}"]

  placement {
    tenancy = "${var.tenancy}"
  }
  network_interfaces {
    associate_public_ip_address = "${var.associate_public_ip_address}"
    delete_on_termination       = true
    security_groups             = ["${concat(list(aws_security_group.lc_security_group.id), var.additional_security_group_ids)}"]
  }
  ebs_optimized = "${var.root_volume_ebs_optimized}"
  block_device_mappings {
    device_name = "${data.aws_ami.ami.root_device_name}"

    ebs {
      encrypted             = "${var.root_volume_ebs_encryption}"
      volume_type           = "${var.root_volume_type}"
      volume_size           = "${var.root_volume_size}"
      delete_on_termination = "${var.root_volume_delete_on_termination}"
    }
  }
  tags = "${var.launch_template_tags}"
  tag_specifications {
    # Instance tags are already handled by the autoscaling group
    resource_type = "volume"

    tags = "${merge(
      var.volume_extra_tags,
      map("key", var.cluster_tag_key, "value", var.cluster_name)
      )
    }"
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
  name_prefix = "${var.cluster_name}"
  description = "Security group for the ${var.cluster_name} launch configuration"
  vpc_id      = "${var.vpc_id}"

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }

  tags = "${merge(var.security_group_tags, map("Name", var.cluster_name))}"
}

resource "aws_security_group_rule" "allow_ssh_inbound_from_cidr_blocks" {
  count       = "${length(var.allowed_ssh_cidr_blocks) >= 1 ? 1 : 0}"
  type        = "ingress"
  from_port   = "${var.ssh_port}"
  to_port     = "${var.ssh_port}"
  protocol    = "tcp"
  cidr_blocks = ["${var.allowed_ssh_cidr_blocks}"]

  security_group_id = "${aws_security_group.lc_security_group.id}"
}

resource "aws_security_group_rule" "allow_ssh_inbound_from_security_group_ids" {
  count                    = "${length(var.allowed_ssh_security_group_ids)}"
  type                     = "ingress"
  from_port                = "${var.ssh_port}"
  to_port                  = "${var.ssh_port}"
  protocol                 = "tcp"
  source_security_group_id = "${element(var.allowed_ssh_security_group_ids, count.index)}"

  security_group_id = "${aws_security_group.lc_security_group.id}"
}

resource "aws_security_group_rule" "allow_all_outbound" {
  type        = "egress"
  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = "${aws_security_group.lc_security_group.id}"
}

# ---------------------------------------------------------------------------------------------------------------------
# THE INBOUND/OUTBOUND RULES FOR THE SECURITY GROUP COME FROM THE VAULT-SECURITY-GROUP-RULES MODULE
# ---------------------------------------------------------------------------------------------------------------------

module "security_group_rules" {
  source = "../vault-security-group-rules"

  security_group_id                    = "${aws_security_group.lc_security_group.id}"
  allowed_inbound_cidr_blocks          = ["${var.allowed_inbound_cidr_blocks}"]
  allowed_inbound_security_group_ids   = ["${var.allowed_inbound_security_group_ids}"]
  allowed_inbound_security_group_count = "${var.allowed_inbound_security_group_count}"

  api_port     = "${var.api_port}"
  cluster_port = "${var.cluster_port}"
}

# ---------------------------------------------------------------------------------------------------------------------
# ATTACH AN IAM ROLE TO EACH EC2 INSTANCE
# We can use the IAM role to grant the instance IAM permissions so we can use the AWS APIs without having to figure out
# how to get our secret AWS access keys onto the box.
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_iam_instance_profile" "instance_profile" {
  name_prefix = "${var.cluster_name}"
  path        = "${var.instance_profile_path}"
  role        = "${aws_iam_role.instance_role.name}"

  # aws_launch_configuration.launch_configuration in this module sets create_before_destroy to true, which means
  # everything it depends on, including this resource, must set it as well, or you'll get cyclic dependency errors
  # when you try to do a terraform destroy.
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_iam_role" "instance_role" {
  name_prefix        = "${var.cluster_name}"
  assume_role_policy = "${data.aws_iam_policy_document.instance_role.json}"

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
  count         = "${var.enable_s3_backend ? 1 : 0}"
  bucket        = "${var.s3_bucket_name}"
  force_destroy = "${var.force_destroy_s3_bucket}"

  tags = "${merge(
    var.s3_bucket_tags,
    map("Description", "Used for secret storage with Vault. DO NOT DELETE this Bucket unless you know what you are doing.")
    )
  }"
}

resource "aws_iam_role_policy" "vault_s3" {
  count  = "${var.enable_s3_backend ? 1 : 0}"
  name   = "vault_s3"
  role   = "${aws_iam_role.instance_role.id}"
  policy = "${element(concat(data.aws_iam_policy_document.vault_s3.*.json, list("")), 0)}"
}

data "aws_iam_policy_document" "vault_s3" {
  count = "${var.enable_s3_backend ? 1 : 0}"

  statement {
    effect  = "Allow"
    actions = ["s3:*"]

    resources = [
      "${aws_s3_bucket.vault_storage.arn}",
      "${aws_s3_bucket.vault_storage.arn}/*",
    ]
  }
}

data "aws_iam_policy_document" "vault_auto_unseal_kms" {
  count  = "${var.enable_auto_unseal ? 1 : 0}"

  statement {
    effect    = "Allow"
    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:DescribeKey"
    ]
    resources = ["${var.auto_unseal_kms_key_arn}"]
  }
}

resource "aws_iam_role_policy" "vault_auto_unseal_kms" {
  count  = "${var.enable_auto_unseal ? 1 : 0}"
  name   = "vault_auto_unseal_kms"
  role   = "${aws_iam_role.instance_role.id}"
  policy = "${element(concat(data.aws_iam_policy_document.vault_auto_unseal_kms.*.json, list("")), 0)}"
}
