package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
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

// To test this on circle ci you need a url set as an environment variable, VAULT_AMI_TEMPLATE_VAR_DOWNLOAD_URL
// which you would also have to set locally if you want to run this test locally.
// The reason is to prevent the actual url from being visible on code and logs
func getUrlFromEnv(t *testing.T) string {
	url := os.Getenv("VAULT_AMI_TEMPLATE_VAR_DOWNLOAD_URL")
	if url == "" {
		t.Fatalf("Please set the environment variable VAULT_AMI_TEMPLATE_VAR_DOWNLOAD_URL.\n")
	}
	return url
}

// Test the Vault auto unseal example by:
//
// 1. Copying the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Building the AMI in the vault-consul-ami example with the given build name
// 3. Deploying a cluster of 1 vault server using the example Terraform code
// 4. Sshing into vault node to initialize the server and check that it booted unsealed
// 5. Increasing the the cluster size to 3 and check that new nodes are unsealed when they boot and join the cluster
func runVaultAutoUnsealTest(t *testing.T, amiId string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_AUTO_UNSEAL_AUTH_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTO_UNSEAL_KMS_KEY_ALIAS: AUTO_UNSEAL_KMS_KEY_ALIAS,
			VAR_VAULT_CLUSTER_SIZE:              1,
		}
		deployCluster(t, amiId, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, WORK_DIR, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		testAutoUnseal(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
	})
}

// Test the Vault enterprise cluster example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name and the enterprise package
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. SSH to a Vault node and make sure you can communicate with the nodes via Consul-managed DNS
// 7. SSH to a Vault node and check if Vault enterprise is installed properly
func runVaultEnterpriseClusterTest(t *testing.T, amiId string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_PRIVATE_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		deployCluster(t, amiId, examplesDir, random.UniqueId(), nil)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, WORK_DIR, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		cluster := initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultUsesConsulForDns(t, cluster)
		checkEnterpriseInstall(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
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

	logger.Logf(t, "Initializing the cluster")
	ssh.CheckSshCommand(t, initialCluster.Leader, "vault operator init")
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

// Check if the enterprise version of consul and vault is installed
func checkEnterpriseInstall(t *testing.T, asgNameOutputVar string, sshUserName string, terratestOptions *terraform.Options, awsRegion string, keyPair *aws.Ec2Keypair) {
	asgName := terraform.OutputRequired(t, terratestOptions, asgNameOutputVar)
	nodeIpAddresses := getIpAddressesOfAsgInstances(t, asgName, awsRegion)

	host := ssh.Host{
		Hostname:    nodeIpAddresses[0],
		SshUserName: sshUserName,
		SshKeyPair:  keyPair.KeyPair,
	}

	maxRetries := 10
	sleepBetweenRetries := 10 * time.Second

	output := retry.DoWithRetry(t, "Check Enterprise Install", maxRetries, sleepBetweenRetries, func() (string, error) {
		out, err := ssh.CheckSshCommandE(t, host, "vault --version")
		if err != nil {
			return "", fmt.Errorf("Error running vault command: %s\n", err)
		}

		return out, nil
	})

	if !strings.Contains(output, "+ent") {
		t.Fatalf("This vault package is not the enterprise version.\n")
	}
}
