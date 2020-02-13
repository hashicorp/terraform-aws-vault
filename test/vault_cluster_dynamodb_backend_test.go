package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

const VAULT_CLUSTER_DYNAMODB_BACKEND_PATH = "examples/vault-dynamodb-backend"

const VAR_DYNAMO_TABLE_NAME = "dynamo_table_name"

// Test the Vault with DynamoDB Backend example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. Connect to the Vault cluster via the ELB
func runVaultWithDynamoBackendClusterTest(t *testing.T, amiId string, awsRegion, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_DYNAMODB_BACKEND_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "log", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		getVaultLogs(t, "vaultClusterWithDynamoBackend", terraformOptions, amiId, awsRegion, sshUserName, keyPair)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		terraformVars := map[string]interface{}{
			VAR_DYNAMO_TABLE_NAME: fmt.Sprintf("vault-dynamo-test-%s", random.UniqueId()),
		}
		deployCluster(t, amiId, awsRegion, examplesDir, random.UniqueId(), terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
	})
}
