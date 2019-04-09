#!/bin/bash
# A script that is meant to be used with the private Vault cluster examples to:
#
# 1. Wait for the Vault server cluster to come up.
# 2. Print out the IP addresses of the Vault servers.
# 3. Print out some example commands you can run against your Vault servers.

set -e

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_NAME="$(basename "$0")"

readonly MAX_RETRIES=30
readonly SLEEP_BETWEEN_RETRIES_SEC=10

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

function assert_is_installed {
  local readonly name="$1"

  if [[ ! $(command -v ${name}) ]]; then
    log_error "The binary '$name' is required by this script but is not installed or in the system's PATH."
    exit 1
  fi
}

function get_optional_terraform_output {
  local readonly output_name="$1"
  terraform output -no-color "$output_name"
}

function get_required_terraform_output {
  local readonly output_name="$1"
  local output_value

  output_value=$(get_optional_terraform_output "$output_name")

  if [[ -z "$output_value" ]]; then
    log_error "Unable to find a value for Terraform output $output_name"
    exit 1
  fi

  echo "$output_value"
}

#
# Usage: join SEPARATOR ARRAY
#
# Joins the elements of ARRAY with the SEPARATOR character between them.
#
# Examples:
#
# join ", " ("A" "B" "C")
#   Returns: "A, B, C"
#
function join {
  local readonly separator="$1"
  shift
  local readonly values=("$@")

  printf "%s$separator" "${values[@]}" | sed "s/$separator$//"
}

function get_all_vault_server_ips {
  local expected_num_vault_servers
  expected_num_vault_servers=$(get_required_terraform_output "vault_cluster_size")

  log_info "Looking up public IP addresses for $expected_num_vault_servers Vault server EC2 Instances."

  local ips
  local i

  for (( i=1; i<="$MAX_RETRIES"; i++ )); do
    ips=($(get_vault_server_ips))
    if [[ "${#ips[@]}" -eq "$expected_num_vault_servers" ]]; then
      log_info "Found all $expected_num_vault_servers public IP addresses!"
      echo "${ips[@]}"
      return
    else
      log_warn "Found ${#ips[@]} of $expected_num_vault_servers public IP addresses. Will sleep for $SLEEP_BETWEEN_RETRIES_SEC seconds and try again."
      sleep "$SLEEP_BETWEEN_RETRIES_SEC"
    fi
  done

  log_error "Failed to find the IP addresses for $expected_num_vault_servers Vault server EC2 Instances after $MAX_RETRIES retries."
  exit 1
}

function wait_for_all_vault_servers_to_come_up {
  local readonly server_ips=($@)

  local expected_num_vault_servers
  expected_num_vault_servers=$(get_required_terraform_output "vault_cluster_size")

  log_info "Waiting for $expected_num_vault_servers Vault servers to come up"

  local server_ip
  for server_ip in "${server_ips[@]}"; do
    wait_for_vault_server_to_come_up "$server_ip"
  done
}

function wait_for_vault_server_to_come_up {
  local readonly server_ip="$1"

  for (( i=1; i<="$MAX_RETRIES"; i++ )); do
    local readonly vault_health_url="https://$server_ip:8200/v1/sys/health"
    log_info "Checking health of Vault server via URL $vault_health_url"

    local response
    local status
    local body

    response=$(curl --show-error --location --insecure --silent --write-out "HTTPSTATUS:%{http_code}" "$vault_health_url" || true)
    status=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS\:.*//g')

    log_info "Got a $status response from Vault server $server_ip with body:\n$body"

    # Response code for the health check endpoint are defined here: https://www.vaultproject.io/api/system/health.html

    if [[ "$status" -eq 200 ]]; then
      log_info "Vault server $server_ip is initialized, unsealed, and active."
      return
    elif [[ "$status" -eq 429 ]]; then
      log_info "Vault server $server_ip is unsealed and in standby mode."
      return
    elif [[ "$status" -eq 501 ]]; then
      log_info "Vault server $server_ip is uninitialized."
      return
    elif [[ "$status" -eq 503 ]]; then
      log_info "Vault server $server_ip is sealed."
      return
    else
      log_info "Vault server $server_ip returned unexpected status code $status. Will sleep for $SLEEP_BETWEEN_RETRIES_SEC seconds and check again."
      sleep "$SLEEP_BETWEEN_RETRIES_SEC"
    fi
  done

  log_error "Did not get a successful response code from Vault server $server_ip after $MAX_RETRIES retries."
  exit 1
}

function get_vault_server_ips {
  local aws_region
  local cluster_tag_key
  local cluster_tag_value
  local instances

  aws_region=$(get_required_terraform_output "aws_region")
  cluster_tag_key=$(get_required_terraform_output "vault_servers_cluster_tag_key")
  cluster_tag_value=$(get_required_terraform_output "vault_servers_cluster_tag_value")

  log_info "Fetching public IP addresses for EC2 Instances in $aws_region with tag $cluster_tag_key=$cluster_tag_value"

  instances=$(aws ec2 describe-instances \
    --region "$aws_region" \
    --filter "Name=tag:$cluster_tag_key,Values=$cluster_tag_value" "Name=instance-state-name,Values=running")

  echo "$instances" | jq -r '.Reservations[].Instances[].PublicIpAddress'
}

function print_instructions {
  local readonly server_ips=($@)
  local server_ip="${server_ips[0]}"

  local ssh_key_name
  ssh_key_name=$(get_required_terraform_output "ssh_key_name")
  ssh_key_name="$ssh_key_name.pem"

  local instructions=()
  instructions+=("\nYour Vault servers are running at the following IP addresses:\n\n${server_ips[@]/#/    }\n")

  instructions+=("To initialize your Vault cluster, SSH to one of the servers and run the init command:\n")
  instructions+=("    ssh -i $ssh_key_name ubuntu@$server_ip")
  instructions+=("    vault operator init")

  instructions+=("\nTo unseal your Vault cluster, SSH to each of the servers and run the unseal command with 3 of the 5 unseal keys:\n")
  for server_ip in "${server_ips[@]}"; do
    instructions+=("    ssh -i $ssh_key_name ubuntu@$server_ip")
    instructions+=("    vault operator unseal (run this 3 times)\n")
  done

  local vault_elb_domain_name
  vault_elb_domain_name=$(get_optional_terraform_output "vault_fully_qualified_domain_name" || true)
  if [[ -z "$vault_elb_domain_name" ]]; then
    vault_elb_domain_name=$(get_optional_terraform_output "vault_elb_dns_name" || true)
  fi

  if [[ -z "$vault_elb_domain_name" ]]; then
    instructions+=("\nOnce your cluster is unsealed, you can read and write secrets by SSHing to any of the servers:\n")
    instructions+=("    ssh -i $ssh_key_name ubuntu@$server_ip")
    instructions+=("    vault login")
    instructions+=("    vault write secret/example value=secret")
    instructions+=("    vault read secret/example")
  else
    instructions+=("\nOnce your cluster is unsealed, you can read and write secrets via the ELB:\n")
    instructions+=("    vault login -address=https://$vault_elb_domain_name")
    instructions+=("    vault write -address=https://$vault_elb_domain_name secret/example value=secret")
    instructions+=("    vault read -address=https://$vault_elb_domain_name secret/example")
  fi

  local instructions_str
  instructions_str=$(join "\n" "${instructions[@]}")

  echo -e "$instructions_str\n"
}

function run {
  assert_is_installed "aws"
  assert_is_installed "jq"
  assert_is_installed "terraform"
  assert_is_installed "curl"

  local server_ips
  server_ips=$(get_all_vault_server_ips)

  wait_for_all_vault_servers_to_come_up "$server_ips"
  print_instructions "$server_ips"
}

run
