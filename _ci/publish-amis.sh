#!/bin/bash
#
# Build the example AMI, copy it to all AWS regions, and make all AMIs public. 
#
# This script is meant to be run in a CircleCI job.
#

set -e

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PACKER_TEMPLATE_PATH="$SCRIPT_DIR/../examples/vault-consul-ami/vault-consul.json"
readonly PACKER_TEMPLATE_DEFAULT_REGION="us-east-1"
readonly AMI_PROPERTIES_FILE="/tmp/ami.properties"

# In CircleCI, every build populates the branch name in CIRCLE_BRANCH...except builds triggered by a new tag, for which
# the CIRCLE_BRANCH env var is empty. We assume tags are only issued against the master branch.
readonly BRANCH_NAME="${CIRCLE_BRANCH:-master}"

readonly PACKER_BUILD_NAME="$1"

if [[ -z "$PACKER_BUILD_NAME" ]]; then
  echo "ERROR: You must pass in the Packer build name as the first argument to this function."
  exit 1
fi

if [[ -z "$PUBLISH_AMI_AWS_ACCESS_KEY_ID" || -z "$PUBLISH_AMI_AWS_SECRET_ACCESS_KEY" ]]; then
  echo "The PUBLISH_AMI_AWS_ACCESS_KEY_ID and PUBLISH_AMI_AWS_SECRET_ACCESS_KEY environment variables must be set to the AWS credentials to use to publish the AMIs."
  exit 1
fi

echo "Checking out branch $BRANCH_NAME to make sure we do all work in a branch and not in detached HEAD state"
git checkout "$BRANCH_NAME"

# We publish the AMIs to a different AWS account, so set those credentials
export AWS_ACCESS_KEY_ID="$PUBLISH_AMI_AWS_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$PUBLISH_AMI_AWS_SECRET_ACCESS_KEY"

# Build the example AMI. Note that we pass in the example TLS files. WARNING! In a production setting, you should
# decrypt or fetch secrets like this when the AMI boots versus embedding them statically into the AMI.
build-packer-artifact \
  --packer-template-path "$PACKER_TEMPLATE_PATH" \
  --build-name "$PACKER_BUILD_NAME" \
  --output-properties-file "$AMI_PROPERTIES_FILE" \
  --var ca_public_key_path="$SCRIPT_DIR/../examples/vault-consul-ami/tls/ca.crt.pem" \
  --var tls_public_key_path="$SCRIPT_DIR/../examples/vault-consul-ami/tls/vault.crt.pem" \
  --var tls_private_key_path="$SCRIPT_DIR/../examples/vault-consul-ami/tls/vault.key.pem"

# Copy the AMI to all regions and make it public in each
source "$AMI_PROPERTIES_FILE"
publish-ami \
  --all-regions \
  --source-ami-id "$ARTIFACT_ID" \
  --source-ami-region "$PACKER_TEMPLATE_DEFAULT_REGION" \
  --markdown-title-text "$PACKER_BUILD_NAME: Latest Public AMIs" \
  --markdown-description-text "**WARNING! Do NOT use these AMIs in a production setting.** They contain TLS certificate files that are publicly available through this repo and using these AMIs in production would represent a serious security risk. The AMIs are meant only to make initial experiments with this module more convenient."
