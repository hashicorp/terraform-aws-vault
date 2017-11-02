output "asg_name" {
  value = "${aws_autoscaling_group.autoscaling_group.name}"
}

output "cluster_tag_key" {
  value = "${var.cluster_tag_key}"
}

output "cluster_tag_value" {
  value = "${var.cluster_name}"
}

output "cluster_size" {
  value = "${aws_autoscaling_group.autoscaling_group.desired_capacity}"
}

output "launch_config_name" {
  value = "${aws_launch_configuration.launch_configuration.name}"
}

output "iam_role_arn" {
  value = "${data.template_file.instance_role_arn.rendered}"
}

output "iam_role_id" {
  value = "${data.template_file.instance_role_id.rendered}"
}

output "security_group_id" {
  value = "${aws_security_group.lc_security_group.id}"
}

output "s3_bucket_arn" {
  value = "${aws_s3_bucket.vault_storage.arn}"
}
