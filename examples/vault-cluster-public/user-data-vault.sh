#!/bin/bash
# This script is meant to be run in the User Data of each EC2 Instance while it's booting. The script uses the
# run-consul script to configure and start Consul in client mode and then the run-vault script to configure and start
# Vault in server mode. Note that this script assumes it's running in an AMI built from the Packer template in
# examples/vault-consul-ami/vault-consul.json.

set -e

# Send the log output from this script to user-data.log, syslog, and the console
# From: https://alestic.com/2010/12/ec2-user-data-output/
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

# The Packer template puts the TLS certs in these file paths
readonly VAULT_TLS_CERT_FILE="/opt/vault/tls/vault.crt.pem"
readonly VAULT_TLS_KEY_FILE="/opt/vault/tls/vault.key.pem"
readonly CA_TLS_CERT_FILE="/opt/vault/tls/ca.crt.pem"

function command_exists {
  local readonly command_name="$1"
  [[ -n "$(command -v $command_name)" ]]
}

function update_certificate_store {
  echo "Adding Vault CA cert file to OS certificate store"

  if $(command_exists "update-ca-certificates"); then
    cp "$CA_TLS_CERT_FILE" /usr/local/share/ca-certificates/
    update-ca-certificates
  elif $(command_exists "update-ca-trust"); then
    update-ca-trust enable
    cp "$CA_TLS_CERT_FILE" /etc/pki/ca-trust/source/anchors/
    update-ca-trust extract
  else
    echo "WARNING: Did not find the update-ca-certificates or update-ca-trust commands. Cannot update OS certificate store. You will have to pass the CA cert file manually to Vault: $CA_TLS_CERT_FILE."
  fi
}

function start_consul_agent {
  local readonly cluster_tag_key="$1"
  local readonly cluster_tag_value="$2"

  echo "Starting Consul agent"
  /opt/consul/bin/run-consul --client --cluster-tag-key "$cluster_tag_key" --cluster-tag-value "$cluster_tag_value"
}

function start_vault_server {
  echo "Starting Vault"
  /opt/vault/bin/run-vault --tls-cert-file "$VAULT_TLS_CERT_FILE"  --tls-key-file "$VAULT_TLS_KEY_FILE"
}

# The variables below are filled in via Terraform interpolation
update_certificate_store
start_consul_agent "${cluster_tag_key}" "${cluster_tag_value}"
start_vault_server
