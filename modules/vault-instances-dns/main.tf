resource "aws_route53_record" "instance_record" {
  count = "${length(var.domain_names)}"

  name = "${var.domain_names[count.index]}"
  type = "A"
  zone_id = "${var.hosted_zone_id}"
  records = ["${var.instance_ips[count.index]}"]
  ttl = "${var.ttl}"
}

resource "aws_route53_record" "srv_record" {
  count = "${var.srv_domain_name == "" ? 0 : 1}"

  name = "_${var.protocol}._tcp.${var.srv_domain_name}"
  type = "SRV"
  zone_id = "${var.hosted_zone_id}"
  # Give equal priority (100) and weight (1) to each instance.
  records = ["${formatlist("100 1 %s %s", var.api_port, var.domain_names)}"]
  ttl = "${var.ttl}"
}
