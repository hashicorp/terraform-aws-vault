# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE SECURITY GROUP RULES THAT CONTROL WHAT TRAFFIC CAN GO IN AND OUT OF A VAULT CLUSTER
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_security_group_rule" "allow_api_inbound_from_cidr_blocks" {
  count       = "${length(var.allowed_inbound_cidr_blocks) >= 1 ? 1 : 0}"
  type        = "ingress"
  from_port   = "${var.api_port}"
  to_port     = "${var.api_port}"
  protocol    = "tcp"
  cidr_blocks = ["${var.allowed_inbound_cidr_blocks}"]

  security_group_id = "${var.security_group_id}"
}

resource "aws_security_group_rule" "allow_api_inbound_from_security_group_ids" {
  count                    = "${var.allowed_inbound_security_group_count}"
  type                     = "ingress"
  from_port                = "${var.api_port}"
  to_port                  = "${var.api_port}"
  protocol                 = "tcp"
  source_security_group_id = "${element(var.allowed_inbound_security_group_ids, count.index)}"

  security_group_id = "${var.security_group_id}"
}

resource "aws_security_group_rule" "allow_cluster_inbound_from_self" {
  type      = "ingress"
  from_port = "${var.cluster_port}"
  to_port   = "${var.cluster_port}"
  protocol  = "tcp"
  self      = true

  security_group_id = "${var.security_group_id}"
}

resource "aws_security_group_rule" "allow_cluster_inbound_from_self_api" {
  type      = "ingress"
  from_port = "${var.api_port}"
  to_port   = "${var.api_port}"
  protocol  = "tcp"
  self      = true

  security_group_id = "${var.security_group_id}"
}
