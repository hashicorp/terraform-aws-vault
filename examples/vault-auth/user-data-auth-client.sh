#!/bin/bash
# This script is meant to be run in the User Data of each EC2 Instance while it's booting. The script uses the
# run-consul script to configure and start Consul in client mode. Note that this script assumes it's running in an AMI
# built from the Packer template in examples/vault-consul-ami/vault-consul.json.

set -e

# Send the log output from this script to user-data.log, syslog, and the console
# From: https://alestic.com/2010/12/ec2-user-data-output/
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

# These variables are passed in via Terraform template interpolation
/opt/consul/bin/run-consul --client --cluster-tag-key "${consul_cluster_tag_key}" --cluster-tag-value "${consul_cluster_tag_value}"


# Retrieve the pkcs7 certificate from instance metadata
# The vault role name is filled by terraform
readonly pkcs7=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7 | tr -d '\n')
readonly data=$(cat <<EOF
{
  "role": "${vault_role_name}",
  "pkcs7": "$pkcs7"
}
EOF
)

#curl --request POST --data "$data" "https://vault.service.consul:8200/v1/auth/aws/login"
