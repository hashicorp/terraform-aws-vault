output "load_balancer_name" {
  value = "${aws_elb.vault.name}"
}

output "load_balancer_dns_name" {
  value = "${aws_elb.vault.dns_name}"
}

output "load_balancer_zone_id" {
  value = "${aws_elb.vault.zone_id}"
}

output "load_balancer_security_group_id" {
  value = "${aws_security_group.vault.id}"
}

output "fully_qualified_domain_name" {
  value = "${element(concat(aws_route53_record.vault_elb.*.fqdn, list("")), 0)}"
}
