#!/bin/bash
# This script is meant to be run in the User Data of each EC2 Instance while it's booting. The script uses the
# run-consul script to configure and start Consul in server mode. Note that this script assumes it's running in an AMI
# built from the Packer template in examples/vault-consul-ami/vault-consul.json.

set -e

# Send the log output from this script to user-data.log, syslog, and the console
# From: https://alestic.com/2010/12/ec2-user-data-output/
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

# These variables are passed in via Terraform template interpolation
#/opt/consul/bin/run-consul --server --cluster-tag-key "${consul_cluster_tag_key}" --cluster-tag-value "${consul_cluster_tag_value}"

#From encryption example

# These variables are passed in via Terraform template interplation
if [[ "${enable_gossip_encryption}" == "true" && ! -z "${gossip_encryption_key}" ]]; then
  # Note that setting the encryption key in plain text here means that it will be readable from the Terraform state file
  # and/or the EC2 API/console. We're doing this for simplicity, but in a real production environment you should pass an
  # encrypted key to Terraform and decrypt it before passing it to run-consul with something like KMS.
  gossip_encryption_configuration="--enable-gossip-encryption --gossip-encryption-key ${gossip_encryption_key}"
fi

if [[ "${enable_rpc_encryption}" == "true" && ! -z "${ca_path}" && ! -z "${cert_file_path}" && ! -z "${key_file_path}" ]]; then
  rpc_encryption_configuration="--enable-rpc-encryption --ca-path ${ca_path} --cert-file-path ${cert_file_path} --key-file-path ${key_file_path}"
fi

# Create acl config including the master token from AWS SecretsManager
TOKEN=$(aws secretsmanager --region "${aws_region}" get-secret-value --secret-id ${consul_token_secret} | jq -r .SecretString)
echo '{"acl": {"tokens": {"master": "'$TOKEN'"}}}' > /opt/consul/config/acl.json

/opt/consul/bin/run-consul --server --datacenter "${consul_cluster_tag_key}" --cluster-tag-key "${consul_cluster_tag_key}" --cluster-tag-value "${consul_cluster_tag_value}" $gossip_encryption_configuration $rpc_encryption_configuration
