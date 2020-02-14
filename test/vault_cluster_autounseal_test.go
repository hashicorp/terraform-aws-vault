package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// This is the alias of a KMS key we have previously created that lives in the
// AWS account where our CI tests run. We have one with the same alias in
// every region. This key is necessary for the test of an Enterprise Vault feature
// called auto unseal. If you wish to run test this locally, replace this with
// the alias of an KMS key you already have on the AWS account you use for running
// your tests or create a new one. Beware that creating an AWS KMS key costs money.
const AUTO_UNSEAL_KMS_KEY_ALIAS = "dedicated-test-key"

const VAULT_AUTO_UNSEAL_AUTH_PATH = "examples/vault-auto-unseal"
const VAR_VAULT_AUTO_UNSEAL_KMS_KEY_ALIAS = "auto_unseal_kms_key_alias"

// Test the Vault auto unseal example by:
//
// 1. Copying the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Building the AMI in the vault-consul-ami example with the given build name
// 3. Deploying a cluster of 1 vault server using the example Terraform code
// 4. Sshing into vault node to initialize the server and check that it booted unsealed
// 5. Increasing the the cluster size to 3 and check that new nodes are unsealed when they boot and join the cluster
func runVaultAutoUnsealTest(t *testing.T, amiId string, awsRegion string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_AUTO_UNSEAL_AUTH_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "log", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		getVaultLogs(t, "vaultAutoUnseal", terraformOptions, amiId, awsRegion, sshUserName, keyPair)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTO_UNSEAL_KMS_KEY_ALIAS: AUTO_UNSEAL_KMS_KEY_ALIAS,
			VAR_VAULT_CLUSTER_SIZE:              1,
			VAR_CONSUL_CLUSTER_NAME:             fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_TAG_KEY:          fmt.Sprintf("consul-test-%s", uniqueId),
		}
		deployCluster(t, amiId, awsRegion, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		testAutoUnseal(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
	})
}

func testAutoUnseal(t *testing.T, asgNameOutputVar string, sshUserName string, terraformOptions *terraform.Options, awsRegion string, keyPair *aws.Ec2Keypair) {
	asgName := terraform.OutputRequired(t, terraformOptions, asgNameOutputVar)
	nodeIpAddresses := getIpAddressesOfAsgInstances(t, asgName, awsRegion)
	logger.Logf(t, fmt.Sprintf("IP ADDRESS OF INSTANCE %s", nodeIpAddresses[0]))
	initialCluster := VaultCluster{
		Leader: ssh.Host{
			Hostname:    nodeIpAddresses[0],
			SshUserName: sshUserName,
			SshKeyPair:  keyPair.KeyPair,
		},
	}

	establishConnectionToCluster(t, initialCluster)
	waitForVaultToBoot(t, initialCluster)

	retry.DoWithRetry(t, "Initializing the cluster", 10, 10*time.Second, func() (string, error) {
		return ssh.CheckSshCommandE(t, initialCluster.Leader, "vault operator init")
	})
	assertStatus(t, initialCluster.Leader, Leader)

	logger.Logf(t, "Increasing the cluster size and running 'terraform apply' again")
	terraformOptions.Vars[VAR_VAULT_CLUSTER_SIZE] = 3
	terraform.Apply(t, terraformOptions)

	logger.Logf(t, "The cluster now should be bigger and the new nodes should boot unsealed (on standby mode already)")
	newCluster := findVaultClusterNodes(t, asgNameOutputVar, sshUserName, terraformOptions, awsRegion, keyPair)
	establishConnectionToCluster(t, newCluster)
	for _, node := range newCluster.Nodes() {
		if node.Hostname != initialCluster.Leader.Hostname {
			assertStatus(t, node, Standby)
		}
	}
}
