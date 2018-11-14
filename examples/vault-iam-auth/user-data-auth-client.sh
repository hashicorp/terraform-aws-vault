#!/bin/bash
# This script is meant to be run in the User Data of each EC2 Instance while it's booting. The script uses the
# run-consul script to configure and start Consul in client mode. Note that this script assumes it's running in an AMI
# built from the Packer template in examples/vault-consul-ami/vault-consul.json.
# It then uses a python script to create a signed request to the AWS STS API and sends this encrypted request
# to the Vault server when authenticating. After login, it reads a secret and exposes it in a simple web server for
# test purposes.

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

# Creating a signed request to AWS Security Token Service (STS) API with header of server ID "vault.service.consul"
# This request is to the method GetCallerIdentity of STS, which answers the question "who am I?"
# This python script creates the STS request, gets the necessary AWS credentials and signs the request with them
# Using python here instead of doing this in bash to take advantage of python's AWS SDK boto3, which facilitates this work a lot
# You can find this script at /examples/vault-consul-ami/auth/sign-request.py
signed_request=$(python /opt/vault/scripts/sign-request.py vault.service.consul)

iam_request_url=$(echo $signed_request | jq -r .iam_request_url)
iam_request_body=$(echo $signed_request | jq -r .iam_request_body)
iam_request_headers=$(echo $signed_request | jq -r .iam_request_headers)

# The role name necessary here is the Vault Role name, not the AWS IAM Role name
# This variable is filled by terraform
data=$(cat <<EOF
{
  "role":"${example_role_name}",
  "iam_http_request_method": "POST",
  "iam_request_url": "$iam_request_url",
  "iam_request_body": "$iam_request_body",
  "iam_request_headers": "$iam_request_headers"
}
EOF
)

# We send this signed request to the Vault server
# And the Vault server will execute this request to validate this origin with AWS
# Retry in case the vault server is still booting and unsealing
# Or in case run-consul running on the background didn't finish yet
login_output=$(retry \
  "curl --fail --request POST --data '$data' https://vault.service.consul:8200/v1/auth/aws/login" \
  "Trying to login to vault")


# If vault cli is installed we can also perform these operations with vault cli
# The VAULT_ADDR environment variable has to be set
# This assumes you have AWS credentials configured in the standard locations
# (environment variables, ~/.aws/credentials, IAM instance profile, or ECS task role, in that order).
# export VAULT_ADDR=https://vault.service.consul:8200
# vault login -method=aws header_value=vault.service.consul role=vault-role-name


# We can then use the client token from the login output once login was successful
token=$(echo $login_output | jq -r .auth.client_token)

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
