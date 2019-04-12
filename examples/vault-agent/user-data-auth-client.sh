#!/bin/bash
# This script is meant to be run in the User Data of each EC2 Instance while it's booting. The script uses the
# run-consul script to configure and start Consul in client mode. Note that this script assumes it's running in an AMI
# built from the Packer template in examples/vault-consul-ami/vault-consul.json.
# It then uses Vault agent to automatically authenticate to the Vault server. After login, Vault agent writes the
# authentication token to a file location, which you can use for your applications.  Note that by default, only the `vault`
# user has access to the file, so you may need to grant the appropriate permissions to your application.
# Finally, this script reads a secret and exposes it in a simple web server for test purposes.

set -e

# Send the log output from this script to user-data.log, syslog, and the console
# From: https://alestic.com/2010/12/ec2-user-data-output/
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

# Log the given message. All logs are written to stderr with a timestamp.
function log {
 local -r message="$1"
 local -r timestamp=$(date +"%Y-%m-%d %H:%M:%S")
 >&2 echo -e "$timestamp $message"
}

# A retry function that attempts to run a command a number of times and returns the output
function retry {
  local -r cmd="$1"
  local -r description="$2"

  for i in $(seq 1 30); do
    log "$description"

    # The boolean operations with the exit status are there to temporarily circumvent the "set -e" at the
    # beginning of this script which exits the script immediatelly for error status while not losing the exit status code
    output=$(eval "$cmd") && exit_status=0 || exit_status=$?
    errors=$(echo "$output") | grep '^{' | jq -r .errors

    log "$output"

    if [[ $exit_status -eq 0 && -n "$output" && -z "$errors" ]]; then
      echo "$output"
      return
    fi
    log "$description failed. Will sleep for 10 seconds and try again."
    sleep 10
  done;

  log "$description failed after 30 attempts."
  exit $exit_status
}

# These variables are passed in via Terraform template interpolation
/opt/consul/bin/run-consul --client --cluster-tag-key "${consul_cluster_tag_key}" --cluster-tag-value "${consul_cluster_tag_value}"

# Start the Vault agent
/opt/vault/bin/run-vault --agent --agent-auth-type iam --agent-auth-role "${example_role_name}"

# Retry and wait for the Vault Agent to write the token out to a file.  This could be
# because the Vault server is still booting and unsealing, or because run-consul
# running on the background didn't finish yet
retry \
  "[[ -s /opt/vault/data/vault-token ]] && echo 'vault token file created'" \
  "waiting for Vault agent to write out token to sink"

# We can then use the client token from the login output once login was successful
token=$(cat /opt/vault/data/vault-token)

# And use the token to perform operations on vault such as reading a secret
# These is being retried because race conditions were causing this to come up null sometimes
response=$(retry \
  "curl --fail -H 'X-Vault-Token: $token' -X GET https://vault.service.consul:8200/v1/secret/example_gruntwork" \
  "Trying to read secret from vault")

# Vault cli alternative:
# export VAULT_TOKEN=$token
# export VAULT_ADDR=https://vault.service.consul:8200
# /opt/vault/bin/vault read secret/example_gruntwork
# Serves the answer in a web server so we can test that this auth client is
# authenticating to vault and fetching data correctly
echo $response | jq -r .data.the_answer > index.html
python -m SimpleHTTPServer 8080 &
