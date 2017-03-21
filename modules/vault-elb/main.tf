# ---------------------------------------------------------------------------------------------------------------------
# THESE TEMPLATES REQUIRE TERRAFORM VERSION 0.8 AND ABOVE
# ---------------------------------------------------------------------------------------------------------------------

terraform {
  required_version = "~> 0.8.0"
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE ELB
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_elb" "vault" {
  name = "${var.name}"

  internal                    = "${var.internal}"
  cross_zone_load_balancing   = "${var.cross_zone_load_balancing}"
  idle_timeout                = "${var.idle_timeout}"
  connection_draining         = "${var.connection_draining}"
  connection_draining_timeout = "${var.connection_draining_timeout}"

  security_groups    = ["${aws_security_group.vault.id}"]
  availability_zones = ["${var.availability_zones}"]
  subnets            = ["${var.subnet_ids}"]

  listener {
    lb_port           = "${var.vault_api_port}"
    lb_protocol       = "TCP"
    instance_port     = "${var.vault_api_port}"
    instance_protocol = "TCP"
  }

  health_check {
    target              = "${var.health_check_protocol}:${var.vault_api_port}${var.health_check_path}"
    interval            = "${var.health_check_interval}"
    healthy_threshold   = "${var.health_check_healthy_threshold}"
    unhealthy_threshold = "${var.health_check_unhealthy_threshold}"
    timeout             = "${var.health_check_timeout}"
  }

  tags {
    Name = "${var.name}"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE SECURITY GROUP THAT CONTROLS WHAT TRAFFIC CAN GO IN AND OUT OF THE ELB
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_security_group" "vault" {
  name        = "${var.name}-elb"
  description = "Security group for the ${var.name} ELB"
  vpc_id      = "${var.vpc_id}"
}

resource "aws_security_group_rule" "allow_inbound_api" {
  type        = "ingress"
  from_port   = "${var.vault_api_port}"
  to_port     = "${var.vault_api_port}"
  protocol    = "tcp"
  cidr_blocks = ["${var.allowed_inbound_cidr_blocks}"]

  security_group_id = "${aws_security_group.vault.id}"
}

resource "aws_security_group_rule" "allow_all_outbound" {
  type        = "egress"
  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = "${aws_security_group.vault.id}"
}

# ---------------------------------------------------------------------------------------------------------------------
# ENABLE PROXY PROTOCOL ON THE LOAD BALANCER
# This carries the information of the original IP address as a header.
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_proxy_protocol_policy" "vault" {
  load_balancer  = "${aws_elb.vault.name}"
  instance_ports = ["${var.vault_api_port}"]
}

# ---------------------------------------------------------------------------------------------------------------------
# ENABLE EXTRA TLS CIPHERS FOR THE VAULT BACKENDS
# It seems that the default TLS ciphers used by AWS are not supported by Go/Vault for HTTPS. Every time the ELB tries
# to do a health check, you get the error:
#
# TLS handshake error from 172.31.79.100:50335: tls: no cipher suite supported by both client and server
#
# Here, we try to add several of the ciphers supposedly supported by go to work around this error
#
# http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/elb-ssl-security-policy.html#ssl-ciphers
# https://golang.org/src/crypto/tls/cipher_suites.go
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_load_balancer_policy" "tls_ciphers" {
  load_balancer_name = "${aws_elb.vault.name}"
  policy_name        = "${var.name}-tls-ciphers"
  policy_type_name   = "SSLNegotiationPolicyType"

  policy_attribute = {
    name  = "ECDHE-RSA-AES128-GCM-SHA256"
    value = "true"
  }

  policy_attribute = {
    name  = "ECDHE-ECDSA-AES128-GCM-SHA256"
    value = "true"
  }

  policy_attribute = {
    name  = "ECDHE-ECDSA-AES256-GCM-SHA384"
    value = "true"
  }

  policy_attribute = {
    name  = "Protocol-TLSv1.2"
    value = "true"
  }
}

resource "aws_load_balancer_backend_server_policy" "tls_ciphers" {
  load_balancer_name = "${aws_elb.vault.name}"
  instance_port      = "${var.vault_api_port}"

  policy_names = [
    "${aws_load_balancer_policy.tls_ciphers.policy_name}",
  ]
}

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONALLY CREATE A ROUTE 53 ENTRY FOR THE ELB
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_route53_record" "vault_elb" {
  count = "${var.create_dns_entry}"

  zone_id = "${var.hosted_zone_id}"
  name    = "${var.domain_name}"
  type    = "A"

  alias {
    name    = "${aws_elb.vault.dns_name}"
    zone_id = "${aws_elb.vault.zone_id}"

    # When set to true, if either none of the ELB's EC2 instances are healthy or the ELB itself is unhealthy,
    # Route 53 routes queries to "other resources." But since we haven't defined any other resources, we'd rather
    # avoid any latency due to switchovers and just wait for the ELB and Vault instances to come back online.
    # For more info, see http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-values-alias.html#rrsets-values-alias-evaluate-target-health
    evaluate_target_health = false
  }
}
