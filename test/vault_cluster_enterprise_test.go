package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

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
func runVaultEnterpriseClusterTest(t *testing.T, amiId string, awsRegion string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_PRIVATE_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "log", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		getVaultLogs(t, "vaultEnterpriseCluster", terraformOptions, amiId, awsRegion, sshUserName, keyPair)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_CONSUL_CLUSTER_NAME:    fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", uniqueId),
		}
		deployCluster(t, amiId, awsRegion, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "initialize_unseal", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		cluster := getInitializedAndUnsealedVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultUsesConsulForDns(t, cluster)
		checkEnterpriseInstall(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
	})
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
