output "asg_name" {
  # This is safe because asg_launch_mechanism will only allow one of aws_autoscaling_group.autoscaling_group.*
  # or aws_autoscaling_group.lt_autoscaling_group.* to be non-empty.
  value = "${join("",concat(aws_autoscaling_group.autoscaling_group.*.name,aws_autoscaling_group.lt_autoscaling_group.*.name))}"
}

output "cluster_tag_key" {
  value = "${var.cluster_tag_key}"
}

output "cluster_tag_value" {
  value = "${var.cluster_name}"
}

output "cluster_size" {
  # This is safe because asg_launch_mechanism will only allow one of aws_autoscaling_group.autoscaling_group.*
  # or aws_autoscaling_group.lt_autoscaling_group.* to be non-empty.
  value = "${element(concat(aws_autoscaling_group.autoscaling_group.*.desired_capacity, aws_autoscaling_group.lt_autoscaling_group.*.desired_capacity), 0)}"
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
