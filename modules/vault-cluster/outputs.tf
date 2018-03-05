output "asg_name" {
  value = "${element(concat(aws_autoscaling_group.autoscaling_group.*.name, list("")), 0)}"
}

output "cluster_tag_key" {
  value = "${var.cluster_tag_key}"
}

output "cluster_tag_value" {
  value = "${var.cluster_name}"
}

output "cluster_size" {
  value = "${var.cluster_size}"
}

output "launch_config_name" {
  value = "${element(concat(aws_launch_configuration.launch_configuration.*.name, list("")), 0)}"
}

output "iam_role_arn" {
  value = "${aws_iam_role.instance_role.arn}"
}

output "iam_role_id" {
  value = "${aws_iam_role.instance_role.id}"
}

output "security_group_id" {
  value = "${aws_security_group.lc_security_group.id}"
}

output "s3_bucket_arn" {
  value = "${join(",", aws_s3_bucket.vault_storage.*.arn)}"
}

# Only available if not using an ASG.

output "instance_ids" {
  value = "${aws_instance.instance.*.id}"
}

output "instance_private_ips" {
  value = "${aws_instance.instance.*.private_ip}"
}

output "instance_public_ips" {
  value = "${aws_instance.instance.*.public_ip}"
}
