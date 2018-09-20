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

# Retrieves the pkcs7 certificate from instance metadata
# The vault role name is filled by terraform
readonly pkcs7=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7 | tr -d '\n')
readonly data=$(cat <<EOF
{
  "role": "${example_role_name}",
  "pkcs7": "$pkcs7"
}
EOF
)

# run-consul is running on the background, so in case it fails we retry
for i in $(seq 1 10); do login_output=$(curl --request POST --data "$data" "https://vault.service.consul:8200/v1/auth/aws/login") && s=0 && break || s=$? && sleep 20; done; (exit $s)

# It is important to note that the default behavior is TOFU(trust on first use)
# So if the pkcs7 certificate gets compromised, attempts to login again will be
# denied unless the client "nonce" returned at the first login is also provided
# Read more at https://www.vaultproject.io/docs/auth/aws.html#client-nonce
#
# nonce=$(echo $login_output | jq -r .auth.metadata.nonce)
# readonly data=$(cat <<EOF
# {
#   "role": "${example_role_name}",
#   "pkcs7": "$pkcs7",
#   "nonce": "$nonce"
# }
# EOF
# )
# curl --request POST --data "$data" "https://vault.service.consul:8200/v1/auth/aws/login"
#
# ==============================================================================
# The output after login will be similar to this:
# {
#   "request_id": "eed334ef-30bc-44a4-2a7f-93ecd7ce23cd",
#   "lease_id": "",
#   "renewable": false,
#   "lease_duration": 0,
#   "data": null,
#   "wrap_info": null,
#   "warnings": [
#     "TTL of \"768h0m0s\" exceeded the effective max_ttl of \"500h0m0s\"; TTL value is capped accordingly"
#   ],
#   "auth": {
#     "client_token": "0ac5b97d-9637-9c03-ce37-77565ed66b8a",
#     "accessor": "f56d56cf-b3a9-d77b-439e-5ea42563a62b",
#     "policies": [
#       "default",
#       "example-policy"
#     ],
#     "token_policies": [
#       "default",
#       "example-policy"
#     ],
#     "metadata": {
#       "account_id": "738755648600",
#       "ami_id": "ami-0a50e8de57a8606a7",
#       "instance_id": "i-0e1c0ef82afa24a7c",
#       "nonce": "d60cf363-eb83-3142-74c3-647445365e32",
#       "region": "eu-west-1",
#       "role": "dev-role",
#       "role_tag_max_ttl": "0s"
#     },
#     "lease_duration": 1800000,
#     "renewable": true,
#     "entity_id": "5051f586-eef5-064e-eca6-768b1de7d19f"
#   }
# }

# We can then use the client token from this output
readonly token=$(echo $login_output | jq -r .auth.client_token)

# And use the token to perform operations on vault such as reading a secret
readonly response=$(curl \
  -H "X-Vault-Token: $token" \
  -X GET \
  https://vault.service.consul:8200/v1/secret/example_gruntwork)

# If vault cli is installed we can also perform this operation by setting the necessary environment variables
# export VAULT_TOKEN=$token
# export VAULT_ADDR=https://vault.service.consul:8200
# /opt/vault/bin/vault read secret/example_gruntwork

# Serves the answer in a web server
echo $response | jq -r .data.the_answer > index.html
python -m SimpleHTTPServer 8080 &
