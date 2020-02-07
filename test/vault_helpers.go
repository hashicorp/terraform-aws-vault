package test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

const REPO_ROOT = "../"
const WORK_DIR = "./"

const ENV_VAR_AWS_REGION = "AWS_DEFAULT_REGION"

const VAR_AMI_ID = "ami_id"
const VAR_VAULT_CLUSTER_NAME = "vault_cluster_name"
const VAR_CONSUL_CLUSTER_NAME = "consul_cluster_name"
const VAR_CONSUL_CLUSTER_TAG_KEY = "consul_cluster_tag_key"
const VAR_SSH_KEY_NAME = "ssh_key_name"
const VAR_VAULT_CLUSTER_SIZE = "vault_cluster_size"
const OUTPUT_VAULT_CLUSTER_ASG_NAME = "asg_name_vault_cluster"

const VAULT_CLUSTER_PUBLIC_OUTPUT_FQDN = "vault_fully_qualified_domain_name"
const VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME = "vault_elb_dns_name"

var UnsealKeyRegex = regexp.MustCompile("^Unseal Key \\d: (.+)$")

const vaultLogFilePath = "/opt/vault/log/vault-journalctl.log"
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

func teardownResources(t *testing.T, examplesDir string) {
	terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
	terraform.Destroy(t, terraformOptions)

	keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)
	aws.DeleteEC2KeyPair(t, keyPair)
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

func deployCluster(t *testing.T, amiId string, awsRegion string, examplesDir string, uniqueId string, terraformVars map[string]interface{}) {
	keyPair := aws.CreateAndImportEC2KeyPair(t, awsRegion, uniqueId)
	test_structure.SaveEc2KeyPair(t, examplesDir, keyPair)

	terraformOptions := &terraform.Options{
		TerraformDir: examplesDir,
		Vars: mergeMaps(terraformVars, map[string]interface{}{
			VAR_AMI_ID:             amiId,
			VAR_VAULT_CLUSTER_NAME: fmt.Sprintf("vault-test-%s", uniqueId),
			VAR_SSH_KEY_NAME:       keyPair.Name,
		}),
		EnvVars: map[string]string{
			ENV_VAR_AWS_REGION: awsRegion,
		},
		// There might be transient errors with the http requests to fetch files
		RetryableTerraformErrors: map[string]string{
			"Error installing provider": "Failed to download terraform package",
		},
	}
	test_structure.SaveTerraformOptions(t, examplesDir, terraformOptions)

	// This function internally retries on allowed errors set in the options
	terraform.InitAndApply(t, terraformOptions)
}

func getVaultLogs(t *testing.T, testId string, terraformOptions *terraform.Options, amiId string, awsRegion string, sshUserName string, keyPair *aws.Ec2Keypair) {
	writeOutVaultLogs(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)

	asgName := terraform.OutputRequired(t, terraformOptions, OUTPUT_VAULT_CLUSTER_ASG_NAME)

	sysLogPath := vaultSyslogPathUbuntu
	if sshUserName == "ec2-user" {
		sysLogPath = vaultSyslogPathAmazonLinux
	}

	instanceIdToFilePathToContents := aws.FetchContentsOfFilesFromAsg(t, awsRegion, sshUserName, keyPair, asgName, true, vaultLogFilePath, sysLogPath)

	require.Len(t, instanceIdToFilePathToContents, vaultClusterSizeInExamples)

	for instanceID, filePathToContents := range instanceIdToFilePathToContents {
		require.Contains(t, filePathToContents, vaultLogFilePath)
		require.Contains(t, filePathToContents, sysLogPath)

		localDestDir := filepath.Join("/tmp/logs/", testId, amiId, instanceID)
		if !files.FileExists(localDestDir) {
			os.MkdirAll(localDestDir, 0755)
		}

		writeLogFile(t, filePathToContents[vaultLogFilePath], filepath.Join(localDestDir, "vault-journalctl.log"))
		writeLogFile(t, filePathToContents[sysLogPath], filepath.Join(localDestDir, "syslog.log"))
	}
}

// Write out the Vault logs from journalctl into a file.  This is mainly used for debugging purposes.
func writeOutVaultLogs(t *testing.T, asgNameOutputVar string, sshUserName string, terraformOptions *terraform.Options, awsRegion string, keyPair *aws.Ec2Keypair) {
	cluster := findVaultClusterNodes(t, asgNameOutputVar, sshUserName, terraformOptions, awsRegion, keyPair)

	for _, node := range cluster.Nodes() {
		output := retry.DoWithRetry(t, "Writing out Vault logs from journalctl to file", 1, 10*time.Second, func() (string, error) {
			return ssh.CheckSshCommandE(t, node, fmt.Sprintf("sudo -u vault mkdir -p /opt/vault/log && journalctl -u vault.service | sudo -u vault tee %s > /dev/null", vaultLogFilePath))
		})
		logger.Logf(t, "Output from journalctl command on %s: %s", node.Hostname, output)
	}

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
		if node.Hostname != "" {
			description := fmt.Sprintf("Trying to establish SSH connection to %s", node.Hostname)
			logger.Logf(t, description)

			maxRetries := 30
			sleepBetweenRetries := 10 * time.Second

			retry.DoWithRetry(t, description, maxRetries, sleepBetweenRetries, func() (string, error) {
				return "", ssh.CheckSshConnectionE(t, node)
			})
		}
	}
}

// Wait until the Vault servers are booted the very first time on the EC2 Instance. As a simple solution, we simply
// wait for the leader to boot and assume if it's up, the other nodes will be too.
func waitForVaultToBoot(t *testing.T, cluster VaultCluster) {
	for _, node := range cluster.Nodes() {
		if node.Hostname != "" {
			logger.Logf(t, "Waiting for Vault to boot the first time on host %s. Expecting it to be in uninitialized status (%d).", node.Hostname, int(Uninitialized))
			assertStatus(t, node, Uninitialized)
		}
	}
}

// Initialize the Vault cluster, filling in the unseal keys in the given vaultCluster struct
func initializeVault(t *testing.T, vaultCluster *VaultCluster) {
	output := retry.DoWithRetry(t, "Initializing the cluster", 10, 10*time.Second, func() (string, error) {
		return ssh.CheckSshCommandE(t, vaultCluster.Leader, "vault operator init")
	})
	vaultCluster.UnsealKeys = parseUnsealKeysFromVaultInitResponse(t, output)
}

// Restart vault
func restartVault(t *testing.T, host ssh.Host) {
	description := fmt.Sprintf("Restarting vault on host %s", host.Hostname)
	retry.DoWithRetry(t, description, 10, 10*time.Second, func() (string, error) {
		return ssh.CheckSshCommandE(t, host, "sudo systemctl restart vault.service")
	})
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
	description := fmt.Sprintf("Unsealing Vault on host %s", host.Hostname)
	retry.DoWithRetryE(t, description, 10, 10*time.Second, func() (string, error) {
		return ssh.CheckSshCommandE(t, host, unsealCommand)
	})
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
