package test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

const REPO_ROOT = "../"

const ENV_VAR_AWS_REGION = "AWS_DEFAULT_REGION"

const VAR_AMI_ID = "ami_id"
const VAR_VAULT_CLUSTER_NAME = "vault_cluster_name"
const VAR_CONSUL_CLUSTER_NAME = "consul_cluster_name"
const VAR_CONSUL_CLUSTER_TAG_KEY = "consul_cluster_tag_key"
const VAR_VAULT_AUTH_SERVER_NAME = "auth_server_name"
const VAR_SSH_KEY_NAME = "ssh_key_name"
const VAR_VAULT_SECRET_NAME = "example_secret"
const VAR_VAULT_IAM_AUTH_ROLE = "example_role_name"
const OUTPUT_VAULT_CLUSTER_ASG_NAME = "asg_name_vault_cluster"
const OUTPUT_AUTH_CLIENT_IP = "auth_client_public_ip"

const VAULT_CLUSTER_PRIVATE_PATH = "examples/vault-cluster-private"
const VAULT_CLUSTER_S3_BACKEND_PATH = "examples/vault-s3-backend"
const VAULT_EC2_AUTH_PATH = "examples/vault-ec2-auth"
const VAULT_IAM_AUTH_PATH = "examples/vault-iam-auth"
const VAULT_CLUSTER_PUBLIC_PATH = REPO_ROOT

const VAULT_CLUSTER_PUBLIC_VAR_CREATE_DNS_ENTRY = "create_dns_entry"
const VAULT_CLUSTER_PUBLIC_VAR_HOSTED_ZONE_DOMAIN_NAME = "hosted_zone_domain_name"
const VAULT_CLUSTER_PUBLIC_VAR_VAULT_DOMAIN_NAME = "vault_domain_name"

const VAULT_CLUSTER_PUBLIC_OUTPUT_FQDN = "vault_fully_qualified_domain_name"
const VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME = "vault_elb_dns_name"

const VAR_ENABLE_S3_BACKEND = "enable_s3_backend"
const VAR_S3_BUCKET_NAME = "s3_bucket_name"
const VAR_FORCE_DESTROY_S3_BUCKET = "force_destroy_s3_bucket"

const AMI_EXAMPLE_PATH = "../examples/vault-consul-ami/vault-consul.json"
const SAVED_AWS_REGION = "AwsRegion"

var UnsealKeyRegex = regexp.MustCompile("^Unseal Key \\d: (.+)$")

const vaultStdOutLogFilePath = "/opt/vault/log/vault-stdout.log"
const vaultStdErrLogFilePath = "/opt/vault/log/vault-error.log"
const vaultSyslogPathUbuntu = "/var/log/syslog"
const vaultSyslogPathAmazonLinux = "/var/log/messages"
const vaultClusterSizeInExamples = 3

type VaultCluster struct {
	Leader     ssh.Host
	Standby1   ssh.Host
	Standby2   ssh.Host
	UnsealKeys []string
}

func (cluster VaultCluster) Nodes() []ssh.Host {
	return []ssh.Host{cluster.Leader, cluster.Standby1, cluster.Standby2}
}

// From: https://www.vaultproject.io/api/system/health.html
type VaultStatus int

const (
	Leader        VaultStatus = 200
	Standby                   = 429
	Uninitialized             = 501
	Sealed                    = 503
)

// Test the Vault private cluster example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. SSH to a Vault node and make sure you can communicate with the nodes via Consul-managed DNS
func runVaultPrivateClusterTest(t *testing.T, packerBuildName string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_PRIVATE_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "setup_ami", func() {
		setupAmi(t, examplesDir, packerBuildName)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		deployCluster(t, examplesDir, random.UniqueId(), nil)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		cluster := initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultUsesConsulForDns(t, cluster)
	})
}

// Test the Vault public cluster example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. Connect to the Vault cluster via the ELB
func runVaultPublicClusterTest(t *testing.T, packerBuildName string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, ".")

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "setup_ami", func() {
		setupAmi(t, examplesDir, packerBuildName)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		terraformVars := map[string]interface{}{
			VAULT_CLUSTER_PUBLIC_VAR_CREATE_DNS_ENTRY:        boolToTerraformVar(false),
			VAULT_CLUSTER_PUBLIC_VAR_HOSTED_ZONE_DOMAIN_NAME: "",
			VAULT_CLUSTER_PUBLIC_VAR_VAULT_DOMAIN_NAME:       "",
		}
		deployCluster(t, examplesDir, random.UniqueId(), terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultViaElb(t, terraformOptions)
	})
}

// Test the Vault with S3 Backend example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. Connect to the Vault cluster via the ELB
func runVaultWithS3BackendClusterTest(t *testing.T, packerBuildName string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_S3_BACKEND_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "logs", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)
		asgName := terraform.OutputRequired(t, terraformOptions, OUTPUT_VAULT_CLUSTER_ASG_NAME)

		sysLogPath := vaultSyslogPathUbuntu
		if sshUserName == "ec2-user" {
			sysLogPath = vaultSyslogPathAmazonLinux
		}

		instanceIdToFilePathToContents := aws.FetchContentsOfFilesFromAsg(t, awsRegion, sshUserName, keyPair, asgName, true, vaultStdOutLogFilePath, vaultStdErrLogFilePath, sysLogPath)

		require.Len(t, instanceIdToFilePathToContents, vaultClusterSizeInExamples)

		for instanceID, filePathToContents := range instanceIdToFilePathToContents {
			require.Contains(t, filePathToContents, vaultStdOutLogFilePath)
			require.Contains(t, filePathToContents, vaultStdErrLogFilePath)
			require.Contains(t, filePathToContents, sysLogPath)

			logger.Logf(t, "Contents of %s on Instance %s:\n\n%s\n", vaultStdOutLogFilePath, instanceID, filePathToContents[vaultStdOutLogFilePath])
			logger.Logf(t, "Contents of %s on Instance %s:\n\n%s\n", vaultStdErrLogFilePath, instanceID, filePathToContents[vaultStdErrLogFilePath])
			logger.Logf(t, "Contents of %s on Instance %s:\n\n%s\n", sysLogPath, instanceID, filePathToContents[sysLogPath])
		}
	})

	test_structure.RunTestStage(t, "setup_ami", func() {
		setupAmi(t, examplesDir, packerBuildName)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_ENABLE_S3_BACKEND:       boolToTerraformVar(true),
			VAR_S3_BUCKET_NAME:          s3BucketName(uniqueId),
			VAR_FORCE_DESTROY_S3_BUCKET: boolToTerraformVar(true),
		}
		deployCluster(t, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		cluster := initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultUsesConsulForDns(t, cluster)
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
func runVaultEnterpriseClusterTest(t *testing.T, packerBuildName string, sshUserName string, vaultDownloadUrl string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_PRIVATE_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "setup_ami", func() {
		awsRegion := aws.GetRandomRegion(t, nil, nil)
		test_structure.SaveString(t, examplesDir, SAVED_AWS_REGION, awsRegion)

		tlsCert := generateSelfSignedTlsCert(t)
		saveTlsCert(t, examplesDir, tlsCert)

		amiId := buildAmiWithDownloadEnv(t, AMI_EXAMPLE_PATH, packerBuildName, tlsCert, awsRegion, vaultDownloadUrl)
		test_structure.SaveAmiId(t, examplesDir, amiId)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		deployCluster(t, examplesDir, random.UniqueId(), nil)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		cluster := initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultUsesConsulForDns(t, cluster)
		checkEnterpriseInstall(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
	})
}

// Test the Vault EC2 authentication example by:
//
// 1. Copying the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Building the AMI in the vault-consul-ami example with the given build name
// 3. Deploying that AMI using the example Terraform code setting an example secret
// 4. Waiting for Vault to boot, then unsealing the server, creating a Vault Role to allow logins from instances with a specific EC2 property and writing the example secret
// 5. Waiting for the client to login, read the secret and launch a simple web server with the contents read
// 6. Making a request to the webserver started by the auth client
func runVaultEC2AuthTest(t *testing.T, packerBuildName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_EC2_AUTH_PATH)
	exampleSecret := "42"

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "setup_ami", func() {
		setupAmi(t, examplesDir, packerBuildName)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTH_SERVER_NAME: fmt.Sprintf("vault-auth-test-%s", uniqueId),
			VAR_VAULT_SECRET_NAME:      exampleSecret,
		}
		deployCluster(t, examplesDir, uniqueId, terraformVars)
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
func runVaultIAMAuthTest(t *testing.T, packerBuildName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_IAM_AUTH_PATH)
	exampleSecret := "42"

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	test_structure.RunTestStage(t, "setup_ami", func() {
		setupAmi(t, examplesDir, packerBuildName)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_VAULT_AUTH_SERVER_NAME: fmt.Sprintf("vault-auth-test-%s", uniqueId),
			VAR_VAULT_IAM_AUTH_ROLE:    fmt.Sprintf("vault-auth-role-test-%s", uniqueId),
			VAR_VAULT_SECRET_NAME:      exampleSecret,
		}
		deployCluster(t, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		testRequestSecret(t, terraformOptions, exampleSecret)
	})
}

func setupAmi(t *testing.T, examplesDir string, packerBuildName string) {
	awsRegion := aws.GetRandomRegion(t, nil, nil)
	test_structure.SaveString(t, examplesDir, SAVED_AWS_REGION, awsRegion)

	tlsCert := generateSelfSignedTlsCert(t)
	saveTlsCert(t, examplesDir, tlsCert)

	amiId := buildAmi(t, AMI_EXAMPLE_PATH, packerBuildName, tlsCert, awsRegion)
	test_structure.SaveAmiId(t, examplesDir, amiId)
}

func teardownResources(t *testing.T, examplesDir string) {
	terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
	terraform.Destroy(t, terraformOptions)

	amiId := test_structure.LoadAmiId(t, examplesDir)
	awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)
	aws.DeleteAmi(t, awsRegion, amiId)

	keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)
	aws.DeleteEC2KeyPair(t, keyPair)

	tlsCert := loadTlsCert(t, examplesDir)
	cleanupTlsCertFiles(tlsCert)
}

// merges map A and B into new map
// if maps are nil, returns an empty map
func mergeMaps(mapA map[string]interface{}, mapB map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}

	if mapA != nil {
		for key, value := range mapA {
			result[key] = value
		}
	}

	if mapB != nil {
		for key, value := range mapB {
			result[key] = value
		}
	}

	return result
}

func deployCluster(t *testing.T, examplesDir string, uniqueId string, terraformVars map[string]interface{}) {
	amiId := test_structure.LoadAmiId(t, examplesDir)
	awsRegion := test_structure.LoadString(t, examplesDir, SAVED_AWS_REGION)

	keyPair := aws.CreateAndImportEC2KeyPair(t, awsRegion, uniqueId)
	test_structure.SaveEc2KeyPair(t, examplesDir, keyPair)

	terraformOptions := &terraform.Options{
		TerraformDir: examplesDir,
		Vars: mergeMaps(terraformVars, map[string]interface{}{
			VAR_AMI_ID:                 amiId,
			VAR_VAULT_CLUSTER_NAME:     fmt.Sprintf("vault-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_NAME:    fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", uniqueId),
			VAR_SSH_KEY_NAME:           keyPair.Name,
		}),
		EnvVars: map[string]string{
			ENV_VAR_AWS_REGION: awsRegion,
		},
	}
	test_structure.SaveTerraformOptions(t, examplesDir, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)
}

// Initialize the Vault cluster and unseal each of the nodes by connecting to them over SSH and executing Vault
// commands. The reason we use SSH rather than using the Vault client remotely is we want to verify that the
// self-signed TLS certificate is properly configured on each server so when you're on that server, you don't
// get errors about the certificate being signed by an unknown party.
func initializeAndUnsealVaultCluster(t *testing.T, asgNameOutputVar string, sshUserName string, terraformOptions *terraform.Options, awsRegion string, keyPair *aws.Ec2Keypair) VaultCluster {
	cluster := findVaultClusterNodes(t, asgNameOutputVar, sshUserName, terraformOptions, awsRegion, keyPair)

	establishConnectionToCluster(t, cluster)
	waitForVaultToBoot(t, cluster)
	initializeVault(t, &cluster)

	assertStatus(t, cluster.Leader, Sealed)
	unsealVaultNode(t, cluster.Leader, cluster.UnsealKeys)
	assertStatus(t, cluster.Leader, Leader)

	assertStatus(t, cluster.Standby1, Sealed)
	unsealVaultNode(t, cluster.Standby1, cluster.UnsealKeys)
	assertStatus(t, cluster.Standby1, Standby)

	assertStatus(t, cluster.Standby2, Sealed)
	unsealVaultNode(t, cluster.Standby2, cluster.UnsealKeys)
	assertStatus(t, cluster.Standby2, Standby)

	return cluster
}

func testRequestSecret(t *testing.T, terraformOptions *terraform.Options, expectedResponse string) {
	instanceIP := terraform.Output(t, terraformOptions, OUTPUT_AUTH_CLIENT_IP)
	url := fmt.Sprintf("http://%s:%s", instanceIP, "8080")

	http_helper.HttpGetWithRetry(t, url, 200, expectedResponse, 30, 10*time.Second)
}

// Find the nodes in the given Vault ASG and return them in a VaultCluster struct
func findVaultClusterNodes(t *testing.T, asgNameOutputVar string, sshUserName string, terraformOptions *terraform.Options, awsRegion string, keyPair *aws.Ec2Keypair) VaultCluster {
	asgName := terraform.Output(t, terraformOptions, asgNameOutputVar)

	nodeIpAddresses := getIpAddressesOfAsgInstances(t, asgName, awsRegion)
	if len(nodeIpAddresses) != 3 {
		t.Fatalf("Expected to get three IP addresses for Vault cluster, but got %d: %v", len(nodeIpAddresses), nodeIpAddresses)
	}

	return VaultCluster{
		Leader: ssh.Host{
			Hostname:    nodeIpAddresses[0],
			SshUserName: sshUserName,
			SshKeyPair:  keyPair.KeyPair,
		},

		Standby1: ssh.Host{
			Hostname:    nodeIpAddresses[1],
			SshUserName: sshUserName,
			SshKeyPair:  keyPair.KeyPair,
		},

		Standby2: ssh.Host{
			Hostname:    nodeIpAddresses[2],
			SshUserName: sshUserName,
			SshKeyPair:  keyPair.KeyPair,
		},
	}
}

// Wait until we can connect to each of the Vault cluster EC2 Instances
func establishConnectionToCluster(t *testing.T, cluster VaultCluster) {
	for _, node := range cluster.Nodes() {
		description := fmt.Sprintf("Trying to establish SSH connection to %s", node.Hostname)
		logger.Logf(t, description)

		maxRetries := 30
		sleepBetweenRetries := 10 * time.Second

		retry.DoWithRetry(t, description, maxRetries, sleepBetweenRetries, func() (string, error) {
			return "", ssh.CheckSshConnectionE(t, node)
		})
	}
}

// Wait until the Vault servers are booted the very first time on the EC2 Instance. As a simple solution, we simply
// wait for the leader to boot and assume if it's up, the other nodes will be too.
func waitForVaultToBoot(t *testing.T, cluster VaultCluster) {
	for _, node := range cluster.Nodes() {
		logger.Logf(t, "Waiting for Vault to boot the first time on host %s. Expecting it to be in uninitialized status (%d).", node.Hostname, int(Uninitialized))
		assertStatus(t, node, Uninitialized)
	}
}

// Initialize the Vault cluster, filling in the unseal keys in the given vaultCluster struct
func initializeVault(t *testing.T, vaultCluster *VaultCluster) {
	logger.Logf(t, "Initializing the cluster")
	output := ssh.CheckSshCommand(t, vaultCluster.Leader, "vault operator init")
	vaultCluster.UnsealKeys = parseUnsealKeysFromVaultInitResponse(t, output)
}

// Parse the unseal keys from the stdout returned from the vault init command.
//
// The format we're expecting is:
//
// Unseal Key 1: Gi9xAX9rFfmHtSi68mYOh0H3H2eu8E77nvRm/0fsuwQB
// Unseal Key 2: ecQjHmaXc79GtwJN/hYWd/N2skhoNgyCmgCfGqRMTPIC
// Unseal Key 3: LEOa/DdZDgLHBqK0JoxbviKByUAgxfm2dwK4y1PX6qED
// Unseal Key 4: ZY87ijsj9/f5fO7ufgr4yhPWU/2ZZM3BGuSQRDFZpwoE
// Unseal Key 5: MAiCaGrtikp4zU4XppC1A8IhKPXRlzj19+a3lcbCAVkF
func parseUnsealKeysFromVaultInitResponse(t *testing.T, vaultInitResponse string) []string {
	lines := strings.Split(vaultInitResponse, "\n")
	if len(lines) < 3 {
		t.Fatalf("Did not find at least three lines of in the vault init stdout: %s", vaultInitResponse)
	}

	// By default, Vault requires 3 unseal keys out of 5, so just parse those first three
	unsealKey1 := parseUnsealKey(t, lines[0])
	unsealKey2 := parseUnsealKey(t, lines[1])
	unsealKey3 := parseUnsealKey(t, lines[2])

	return []string{unsealKey1, unsealKey2, unsealKey3}
}

// Generate a unique name for an S3 bucket. Note that S3 bucket names must be globally unique and that only lowercase
// alphanumeric characters and hyphens are allowed.
func s3BucketName(uniqueId string) string {
	return strings.ToLower(fmt.Sprintf("vault-module-test-%s", uniqueId))
}

// SSH to a Vault node and make sure that is properly configured to use Consul for DNS so that the vault.service.consul
// domain name works.
func testVaultUsesConsulForDns(t *testing.T, cluster VaultCluster) {
	// Pick any host, it shouldn't matter
	host := cluster.Standby1

	command := "vault status -address=https://vault.service.consul:8200"
	description := fmt.Sprintf("Checking that the Vault server at %s is properly configured to use Consul for DNS: %s", host.Hostname, command)
	logger.Logf(t, description)

	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	out, err := retry.DoWithRetryE(t, description, maxRetries, sleepBetweenRetries, func() (string, error) {
		return ssh.CheckSshCommandE(t, host, command)
	})

	logger.Logf(t, "Output from command vault status call to vault.service.consul: %s", out)
	if err != nil {
		t.Fatalf("Failed to run vault command with vault.service.consul URL due to error: %v", err)
	}
}

// Use the Vault client to connect to the Vault via the ELB, via the public DNS entry, and make sure it works without
// Vault or TLS errors
func testVaultViaElb(t *testing.T, terraformOptions *terraform.Options) {
	domainName := getElbDomainName(t, terraformOptions)
	description := fmt.Sprintf("Testing Vault via ELB at domain name %s", domainName)
	logger.Logf(t, description)

	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	vaultClient := createVaultClient(t, domainName)

	out := retry.DoWithRetry(t, description, maxRetries, sleepBetweenRetries, func() (string, error) {
		isInitialized, err := vaultClient.Sys().InitStatus()
		if err != nil {
			return "", err
		}
		if isInitialized {
			return "Successfully verified that Vault cluster is initialized via ELB!", nil
		} else {
			return "", errors.New("Expected Vault cluster to be initialized, but ELB reports it is not.")
		}
	})

	logger.Logf(t, out)
}

// Get the ELB domain name
func getElbDomainName(t *testing.T, terraformOptions *terraform.Options) string {
	return terraform.OutputRequired(t, terraformOptions, VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME)
}

// Create a Vault client configured to talk to Vault running at the given domain name
func createVaultClient(t *testing.T, domainName string) *api.Client {
	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("https://%s", domainName)

	// The TLS cert we are using in this test does not have the ELB DNS name in it, so disable the TLS check
	clientTLSConfig := config.HttpClient.Transport.(*http.Transport).TLSClientConfig
	clientTLSConfig.InsecureSkipVerify = true

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	return client
}

// Unseal the given Vault server using the given unseal keys
func unsealVaultNode(t *testing.T, host ssh.Host, unsealKeys []string) {
	unsealCommands := []string{}
	for _, unsealKey := range unsealKeys {
		unsealCommands = append(unsealCommands, fmt.Sprintf("vault operator unseal %s", unsealKey))
	}

	unsealCommand := strings.Join(unsealCommands, " && ")

	logger.Logf(t, "Unsealing Vault on host %s", host.Hostname)
	ssh.CheckSshCommand(t, host, unsealCommand)
}

// Parse an unseal key from a single line of the stdout of the vault init command, which should be of the format:
//
// Unseal Key 1: Gi9xAX9rFfmHtSi68mYOh0H3H2eu8E77nvRm/0fsuwQB
func parseUnsealKey(t *testing.T, str string) string {
	matches := UnsealKeyRegex.FindStringSubmatch(str)
	if len(matches) != 2 {
		t.Fatalf("Unexpected format for unseal key: %s", str)
	}
	return matches[1]
}

// There is a bug with Terraform where if you try to pass a boolean as a -var parameter (e.g. -var foo=true), you get
// a strconv.ParseInt error. To work around it, we convert our booleans to the equivalent int.
func boolToTerraformVar(val bool) int {
	if val {
		return 1
	} else {
		return 0
	}
}

// Check that the Vault node at the given host has the given
func assertStatus(t *testing.T, host ssh.Host, expectedStatus VaultStatus) {
	description := fmt.Sprintf("Check that the Vault node %s has status %d", host.Hostname, int(expectedStatus))
	logger.Logf(t, description)

	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	out := retry.DoWithRetry(t, description, maxRetries, sleepBetweenRetries, func() (string, error) {
		return checkStatus(t, host, expectedStatus)
	})

	logger.Logf(t, out)
}

// Delete the temporary self-signed cert files we created
func cleanupTlsCertFiles(tlsCert TlsCert) {
	os.Remove(tlsCert.CAPublicKeyPath)
	os.Remove(tlsCert.PrivateKeyPath)
	os.Remove(tlsCert.PublicKeyPath)
}

// Check the status of the given Vault node and ensure it matches the expected status. Note that we use curl to do the
// status check so we can ensure that TLS certificates work for curl (and not just the Vault client).
func checkStatus(t *testing.T, host ssh.Host, expectedStatus VaultStatus) (string, error) {
	curlCommand := "curl -s -o /dev/null -w '%{http_code}' https://127.0.0.1:8200/v1/sys/health"
	logger.Logf(t, "Using curl to check status of Vault server %s: %s", host.Hostname, curlCommand)

	output, err := ssh.CheckSshCommandE(t, host, curlCommand)
	if err != nil {
		return "", err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return "", err
	}

	if status == int(expectedStatus) {
		return fmt.Sprintf("Got expected status code %d", status), nil
	} else {
		return "", fmt.Errorf("Expected status code %d for host %s, but got %d", int(expectedStatus), host.Hostname, status)
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
