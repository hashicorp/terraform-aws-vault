#!/bin/bash
# This script is meant to be run in the User Data of each EC2 Instance while it's booting. The script uses the
# run-consul script to configure and start Consul in client mode and then the run-vault script to configure and start
# Vault in server mode. The script then applies the enterprise license (passed by terraform) and checks the server
# status to verify that it is unsealed. Note that this script assumes it's running in an AMI built from the Packer
# template in examples/vault-consul-ami/vault-consul.json.


set -e

# Send the log output from this script to user-data.log, syslog, and the console
# From: https://alestic.com/2010/12/ec2-user-data-output/
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1


# Log the given message at the given level. All logs are written to stderr with a timestamp.
function log {
 local readonly message="$1"
 local readonly timestamp=$(date +"%Y-%m-%d %H:%M:%S")
 >&2 echo -e "$timestamp $message"
}

# A retry function that attempts to run a command a number of times and returns the output
function retry {
  local readonly cmd=$1
  local readonly description=$2

  for i in $(seq 1 30); do
    log "$description"

    # The boolean operations with the exit status are there to temporarily circumvent the "set -e" at the
    # beginning of this script which exits the script immediatelly for error status while not losing the exit status code
    output=$(eval "$cmd") && exit_status=0 || exit_status=$?
    if [[ $exit_status -eq 0 ]]; then
      echo "$output"
      return
    fi
    log "$description failed. Will sleep for 10 seconds and try again."
    sleep 10
  done;

  log "$description failed after 30 attempts."
  exit $exit_status
}

# The Packer template puts the TLS certs in these file paths
readonly VAULT_TLS_CERT_FILE="/opt/vault/tls/vault.crt.pem"
readonly VAULT_TLS_KEY_FILE="/opt/vault/tls/vault.key.pem"

# The cluster_tag variables below are filled in via Terraform interpolation
/opt/consul/bin/run-consul --client --cluster-tag-key "${consul_cluster_tag_key}" --cluster-tag-value "${consul_cluster_tag_value}"
/opt/vault/bin/run-vault \
  --tls-cert-file "$VAULT_TLS_CERT_FILE" \
  --tls-key-file "$VAULT_TLS_KEY_FILE" \
  --enable-auto-unseal \
  --auto-unseal-kms-key-id "${kms_key_id}" \
  --auto-unseal-kms-key-region "${aws_region}"

# Initializes the vault server
# Retries this a number of times because run-vault is running on the background and
# we need to wait for this to finish.
# This userdata file is present in all instances in the cluster, but the code below
# will only succeed in the first one that runs it, the others will fail with "the server has
# already been initialized" and the script will be interrupted. This is not an
# issue, and, for the purpose of this example, it is simpler than reorganizing the
# code in such a way that the nodes receive different user data files.
server_output=$(retry \
  "/opt/vault/bin/vault operator init" \
  "Initializing Vault server")

# The code below will only run in the one node that succeeded in initializing the
# server first, the others will exit at the line above due to the `set -e` at the
# beginning of this script. Again, not an issue.

# Exports the client token environment variable necessary for running the vault
# write commands
export VAULT_TOKEN=$(echo "$server_output" | head -n 7 | tail -n 1 | awk '{ print $4; }')

# Applies the enterprise license to the vault cluster
# This value is being passed by terraform
/opt/vault/bin/vault write /sys/license "text=${vault_enterprise_license_key}"

# Vault now should be unsealed! If it still shows as sealed on some of the instances
# in the cluster, vault might have to be restarted with
# sudo supervisorctl restart vault
/opt/vault/bin/vault status
