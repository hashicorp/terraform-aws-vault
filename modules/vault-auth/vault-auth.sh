#!/bin/bash
# This script can be used to setup vault ec2 authentication on a vault server or authenticate an instance to vault.

set -e

readonly DEFAULT_INSTALL_PATH="/opt/vault"

readonly DEFAULT_AUTH_TYPE="ec2"
readonly DEFAULT_ROLE_NAME="dev-role"
readonly DEFAULT_POLICY_NAME="dev"
readonly DEFAULT_MAX_TTL="500h"
readonly DEFAULT_VAULT_URL="https://vault.service.consul:8200"


function print_usage {
  echo
  echo "Usage: install-vault [OPTIONS]"
  echo
  echo "This script can be used to setup vault ec2 authentication on a vault server or authenticate an instance to vault."
  echo
  echo "Options:"
  echo
  echo -e "  --server\t\tConfigure a vault server to enable authentication (after it has been initiated and unsealed)."
  echo -e "  --client\t\tAuthenticate instance to a vault server."
  echo
  echo "Example:"
  echo
  echo "  vault-auth --server"
  #TODO describe args
}

function log {
  local readonly level="$1"
  local readonly message="$2"
  local readonly timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  >&2 echo -e "${timestamp} [${level}] [$SCRIPT_NAME] ${message}"
}

function log_info {
  local readonly message="$1"
  log "INFO" "$message"
}

function log_warn {
  local readonly message="$1"
  log "WARN" "$message"
}

function log_error {
  local readonly message="$1"
  log "ERROR" "$message"
}

function assert_not_empty {
  local readonly arg_name="$1"
  local readonly arg_value="$2"

  if [[ -z "$arg_value" ]]; then
    log_error "The value for '$arg_name' cannot be empty"
    print_usage
    exit 1
  fi
}

function assert_either_or {
  local readonly arg1_name="$1"
  local readonly arg1_value="$2"
  local readonly arg2_name="$3"
  local readonly arg2_value="$4"

  if [[ -z "$arg1_value" && -z "$arg2_value" ]]; then
    log_error "Either the value for '$arg1_name' or '$arg2_name' must be passed, both cannot be empty"
    print_usage
    exit 1
  fi
}


function configure_auth {
  # Attention, vault needs to be running and unsealed already

  local readonly auth_type="$1"
  local readonly role_name="$2"
  local readonly policy_name="$3"
  local readonly max_ttl="$4"

  vault auth enable aws

  # Create policy
  vault policy write "$policy_name" -<<EOF
path "secret/*" {
	capabilities = ["create", "read"]
}
EOF
  #TODO allow overwriting of this simple example policy, or maybe just the backend path

  # Configure authentication
  vault write \
    auth/aws/role/$role_name \
    auth_type=$auth_type \
    policies=$policy_name \
    max_ttl=$max_ttl \
    #TODO decide how to set all settings for ec2 metadata to be used as criteria for auth
    #example with ami id
    bound_ami_id=ami-0a50e8de57a8606a7
}

function login {
  local readonly role_name="$1"
  local readonly vault_server_ip="$2"
  local readonly vault_url="$3"
  local readonly pkcs7=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7 | tr -d '\n')
  local readonly data=$(cat <<EOF
{
  "role": "$role_name",
  "pkcs7": "$pkcs7"
}
EOF
)

  consul join vault_server_ip
  curl --request POST --data "$data" "$vault_url/v1/auth/aws/login"
  #TODO decide what to do with nonce & token
}

function auth {
  local server=""
  local client=""
  local auth_type="$DEFAULT_AUTH_TYPE"
  local role_name="$DEFAULT_ROLE_NAME"
  local policy_name="$DEFAULT_POLICY_NAME"
  local max_ttl="$DEFAULT_MAX_TTL"
  local vault_server_ip=""
  local vault_url="$DEFAULT_VAULT_URL"

  while [[ $# > 0 ]]; do
    local key="$1"

    case "$key" in
      --server)
        server="true"
        shift
        ;;
      --client)
        server="client"
        shift
        ;;
      --auth-type)
        auth_type="$2"
        shift
        ;;
      --role-name)
        role_name="$2"
        shift
        ;;
      --policy-name)
        policy_name="$2"
        shift
        ;;
      --max-ttl)
        max_ttl="$2"
        shift
        ;;
      --vault-server-ip)
        vault_server_ip="$2"
        shift
        ;;
      --vault-url)
        vault_url="$2"
        shift
        ;;
      --help)
        print_usage
        exit
        ;;
      *)
        log_error "Unrecognized argument: $key"
        print_usage
        exit 1
        ;;
    esac

    shift
  done

  assert_either_or "--server" "$server" "--client" "$client"

  if [[ "$server" == "true" ]]; then
    log_info "Configuring authentication on Vault server"
    configure_auth "$auth_type" "$role_name" "$policy_name" "$max_ttl"
  elif [[ "$client" == "true" ]]; then
    assert_not_empty $vault_server_ip
    log_info "Authenticating to Vault server"
    configure_auth "$role_name" "$vault_server_ip" "$vault_url"
  fi

}

auth "$@"
