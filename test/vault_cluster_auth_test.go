package test

import (
	"fmt"
	"os"
	"path/filepath"

	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/files"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

const VAULT_EC2_AUTH_PATH = "examples/vault-ec2-auth"
const VAULT_IAM_AUTH_PATH = "examples/vault-iam-auth"
const VAULT_AGENT_PATH = "examples/vault-agent"

const VAR_VAULT_AUTH_SERVER_NAME = "auth_server_name"
const VAR_VAULT_SECRET_NAME = "example_secret"
const VAR_VAULT_IAM_AUTH_ROLE = "example_role_name"

const OUTPUT_AUTH_CLIENT_IP = "auth_client_public_ip"
const OUTPUT_AUTH_CLIENT_INSTANCE_ID = "auth_client_instance_id"

// Test the Vault EC2 authentication example by:
//
// 1. Copying the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Building the AMI in the vault-consul-ami example with the given build name
// 3. Deploying that AMI using the example Terraform code setting an example secret
// 4. Waiting for Vault to boot, then unsealing the server, creating a Vault Role to allow logins from instances with a specific EC2 property and writing the example secret
// 5. Waiting for the client to login, read the secret and launch a simple web server with the contents read
// 6. Making a request to the webserver started by the auth client
func runVaultEC2AuthTest(t *testing.T, amiId string, awsRegion string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_EC2_AUTH_PATH)
	exampleSecret := "42"

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "log", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		getSyslogs(t, terraformOptions, amiId, awsRegion, "vaultEc2Auth")
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTH_SERVER_NAME: fmt.Sprintf("vault-auth-test-%s", uniqueId),
			VAR_VAULT_SECRET_NAME:      exampleSecret,
			VAR_CONSUL_CLUSTER_NAME:    fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", uniqueId),
		}
		deployCluster(t, amiId, awsRegion, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		testRequestSecret(t, terraformOptions, exampleSecret)
	})
}

// Test the Vault IAM authentication example by:
//
// 1. Copying the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Building the AMI in the vault-consul-ami example with the given build name
// 3. Deploying that AMI using the example Terraform code setting an example secret
// 4. Waiting for Vault to boot, then unsealing the server, creating a Vault Role to allow logins from resources with a specific AWS IAM Role and writing the example secret
// 5. Waiting for the client to login, read the secret and launch a simple web server with the contents read
// 6. Making a request to the webserver started by the auth client
func runVaultIAMAuthTest(t *testing.T, amiId string, awsRegion string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_IAM_AUTH_PATH)
	exampleSecret := "42"

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "log", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		getSyslogs(t, terraformOptions, amiId, awsRegion, "vaultIamAuth")
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTH_SERVER_NAME: fmt.Sprintf("vault-auth-test-%s", uniqueId),
			VAR_VAULT_IAM_AUTH_ROLE:    fmt.Sprintf("vault-auth-role-test-%s", uniqueId),
			VAR_VAULT_SECRET_NAME:      exampleSecret,
			VAR_CONSUL_CLUSTER_NAME:    fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", uniqueId),
		}
		deployCluster(t, amiId, awsRegion, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		testRequestSecret(t, terraformOptions, exampleSecret)
	})
}

// Test the Vault agent authentication example by:
//
// 1. Copying the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Building the AMI in the vault-consul-ami example with the given build name
// 3. Deploying that AMI using the example Terraform code setting an example secret
// 4. Waiting for Vault to boot, then unsealing the server, creating a Vault Role to allow logins from resources with a specific AWS IAM Role and writing the example secret
// 5. Waiting for the client to login, read the secret and launch a simple web server with the contents read
// 6. Making a request to the webserver started by the auth client
func runVaultAgentTest(t *testing.T, amiId string, awsRegion string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_AGENT_PATH)
	exampleSecret := "42"

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "log", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		getSyslogs(t, terraformOptions, amiId, awsRegion, "vaultAgentAuth")
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTH_SERVER_NAME: fmt.Sprintf("vault-auth-test-%s", uniqueId),
			VAR_VAULT_IAM_AUTH_ROLE:    fmt.Sprintf("vault-auth-role-test-%s", uniqueId),
			VAR_VAULT_SECRET_NAME:      exampleSecret,
			VAR_CONSUL_CLUSTER_NAME:    fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", uniqueId),
		}
		deployCluster(t, amiId, awsRegion, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		testRequestSecret(t, terraformOptions, exampleSecret)
	})
}

func testRequestSecret(t *testing.T, terraformOptions *terraform.Options, expectedResponse string) {
	instanceIP := terraform.Output(t, terraformOptions, OUTPUT_AUTH_CLIENT_IP)
	url := fmt.Sprintf("http://%s:%s", instanceIP, "8080")

	http_helper.HttpGetWithRetry(t, url, 200, expectedResponse, 30, 10*time.Second)
}

func getSyslogs(t *testing.T, terraformOptions *terraform.Options, amiId string, awsRegion string, testName string) {
	clientInstanceId := terraform.OutputRequired(t, terraformOptions, OUTPUT_AUTH_CLIENT_INSTANCE_ID)
	serverAsgName := terraform.OutputRequired(t, terraformOptions, OUTPUT_VAULT_CLUSTER_ASG_NAME)

	serverLogs, err := aws.GetSyslogForInstancesInAsgE(t, serverAsgName, awsRegion)
	if err != nil {
		logger.Logf(t, fmt.Sprintf("Error getting vault server syslog: %s", err.Error()))
	}

	clientLog, err := aws.GetSyslogForInstanceE(t, clientInstanceId, awsRegion)
	if err != nil {
		logger.Logf(t, fmt.Sprintf("Error getting vault client syslog: %s", err.Error()))
	}

	localDestDir := filepath.Join("/tmp/logs", testName, amiId)
	if !files.FileExists(localDestDir) {
		os.MkdirAll(localDestDir, 0755)
	}

	for id, buf := range serverLogs {
		writeLogFile(t, buf, filepath.Join(localDestDir, fmt.Sprintf("vault-server-%s-syslog.log", id)))
	}
	writeLogFile(t, clientLog, filepath.Join(localDestDir, "auth-client-syslog.log"))
}
