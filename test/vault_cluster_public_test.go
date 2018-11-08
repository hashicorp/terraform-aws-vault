package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/test-structure"
)

const VAULT_CLUSTER_PUBLIC_PATH = REPO_ROOT

const VAULT_CLUSTER_PUBLIC_VAR_CREATE_DNS_ENTRY = "create_dns_entry"
const VAULT_CLUSTER_PUBLIC_VAR_HOSTED_ZONE_DOMAIN_NAME = "hosted_zone_domain_name"
const VAULT_CLUSTER_PUBLIC_VAR_VAULT_DOMAIN_NAME = "vault_domain_name"

// Test the Vault public cluster example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. Connect to the Vault cluster via the ELB
func runVaultPublicClusterTest(t *testing.T, amiId string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, ".")

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		terraformVars := map[string]interface{}{
			VAULT_CLUSTER_PUBLIC_VAR_CREATE_DNS_ENTRY:        boolToTerraformVar(false),
			VAULT_CLUSTER_PUBLIC_VAR_HOSTED_ZONE_DOMAIN_NAME: "",
			VAULT_CLUSTER_PUBLIC_VAR_VAULT_DOMAIN_NAME:       "",
		}
		deployCluster(t, amiId, examplesDir, random.UniqueId(), terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, WORK_DIR, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultViaElb(t, terraformOptions)
	})
}
