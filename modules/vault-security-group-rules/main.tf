# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE SECURITY GROUP RULES THAT CONTROL WHAT TRAFFIC CAN GO IN AND OUT OF A VAULT CLUSTER
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_security_group_rule" "allow_api_inbound_from_cidr_blocks" {
  type        = "ingress"
  from_port   = "${var.api_port}"
  to_port     = "${var.api_port}"
  protocol    = "tcp"
  cidr_blocks = ["${var.allowed_inbound_cidr_blocks}"]

  security_group_id = "${var.security_group_id}"
}

resource "aws_security_group_rule" "allow_cluster_inbound_from_self" {
  type        = "ingress"
  from_port   = "${var.cluster_port}"
  to_port     = "${var.cluster_port}"
  protocol    = "tcp"
  self        = true

  security_group_id = "${var.security_group_id}"
}
