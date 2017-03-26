package test

import (
	"github.com/gruntwork-io/terratest"
	"testing"
	"os"
	terralog "github.com/gruntwork-io/terratest/log"
	"log"
	"github.com/gruntwork-io/terratest/util"
	"time"
	"fmt"
	"path/filepath"
	"regexp"
	"github.com/gruntwork-io/terratest/ssh"
	"strconv"
	"strings"
	"github.com/hashicorp/vault/api"
	"net/http"
	"errors"
)

const REPO_ROOT = "../"

const VAR_AWS_REGION = "aws_region"
const VAR_AMI_ID = "ami_id"
const VAR_S3_BUCKET_NAME = "s3_bucket_name"
const VAR_VAULT_CLUSTER_NAME = "vault_cluster_name"
const VAR_CONSUL_CLUSTER_NAME = "consul_cluster_name"
const VAR_CONSUL_CLUSTER_TAG_KEY = "consul_cluster_tag_key"
const VAR_SSH_KEY_NAME = "ssh_key_name"
const VAR_FORCE_DESTROY_S3_BUCKET = "force_destroy_s3_bucket"
const OUTPUT_VAULT_CLUSTER_ASG_NAME = "asg_name_vault_cluster"

const VAULT_CLUSTER_PRIVATE_PATH = "examples/vault-cluster-private"
const VAULT_CLUSTER_PUBLIC_PATH = "examples/vault-cluster-public"

const VAULT_CLUSTER_PUBLIC_VAR_CREATE_DNS_ENTRY = "create_dns_entry"
const VAULT_CLUSTER_PUBLIC_VAR_HOSTED_ZONE_DOMAIN_NAME = "hosted_zone_domain_name"
const VAULT_CLUSTER_PUBLIC_VAR_VAULT_DOMAIN_NAME = "vault_domain_name"

const VAULT_CLUSTER_PUBLIC_OUTPUT_FQDN = "vault_fully_qualified_domain_name"
const VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME = "vault_elb_dns_name"

const AMI_EXAMPLE_PATH = "../examples/vault-consul-ami/vault-consul.json"

var UnsealKeyRegex = regexp.MustCompile("^Unseal Key \\d: (.+)$")

type VaultCluster struct {
	Leader  	ssh.Host
	Standby1	ssh.Host
	Standby2  	ssh.Host
	UnsealKeys	[]string
}

func (cluster VaultCluster) Nodes() []ssh.Host {
	return []ssh.Host{cluster.Leader, cluster.Standby1, cluster.Standby2}
}

// From: https://www.vaultproject.io/api/system/health.html
type VaultStatus int
const (
	Leader VaultStatus = 200
	Standby            = 429
	Uninitialized      = 501
	Sealed             = 503
)

// Test the Vault private cluster example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 5. SSH to a Vault node and make sure you can communicate with the nodes via Consul-managed DNS
func runVaultPrivateClusterTest(t *testing.T, testName string, packerBuildName string, sshUserName string) {
	rootTempPath := copyRepoToTempFolder(t, REPO_ROOT)
	defer os.RemoveAll(rootTempPath)

	resourceCollection := createBaseRandomResourceCollection(t)
	terratestOptions := createBaseTerratestOptions(t, testName, filepath.Join(rootTempPath, VAULT_CLUSTER_PRIVATE_PATH), resourceCollection)
	defer terratest.Destroy(terratestOptions, resourceCollection)

	tlsCert := generateSelfSignedTlsCert(t, testName)
	defer cleanupTlsCertFiles(tlsCert)

	logger := terralog.NewLogger(testName)
	amiId := buildAmi(t, AMI_EXAMPLE_PATH, packerBuildName, tlsCert, resourceCollection, logger)

	terratestOptions.Vars = map[string]interface{} {
		VAR_AMI_ID: amiId,
		VAR_AWS_REGION: resourceCollection.AwsRegion,
		VAR_S3_BUCKET_NAME: s3BucketName(resourceCollection),
		VAR_VAULT_CLUSTER_NAME: fmt.Sprintf("vault-test-%s", resourceCollection.UniqueId),
		VAR_CONSUL_CLUSTER_NAME: fmt.Sprintf("consul-test-%s", resourceCollection.UniqueId),
		VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", resourceCollection.UniqueId),
		VAR_SSH_KEY_NAME: resourceCollection.KeyPair.Name,
		VAR_FORCE_DESTROY_S3_BUCKET: boolToTerraformVar(true),
	}

	deploy(t, terratestOptions)
	cluster := initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terratestOptions, resourceCollection, logger)
	testVaultUsesConsulForDns(t, cluster, logger)
}

// Test the Valut public cluster example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. Connect to the Vault cluster via the ELB
func runVaultPublicClusterTest(t *testing.T, testName string, packerBuildName string, sshUserName string) {
	rootTempPath := copyRepoToTempFolder(t, REPO_ROOT)
	defer os.RemoveAll(rootTempPath)

	logger := terralog.NewLogger(testName)
	resourceCollection := createBaseRandomResourceCollection(t)
	terratestOptions := createBaseTerratestOptions(t, testName, filepath.Join(rootTempPath, VAULT_CLUSTER_PUBLIC_PATH), resourceCollection)
	defer terratest.Destroy(terratestOptions, resourceCollection)

	tlsCert := generateSelfSignedTlsCert(t, testName)
	defer cleanupTlsCertFiles(tlsCert)

	amiId := buildAmi(t, AMI_EXAMPLE_PATH, packerBuildName, tlsCert, resourceCollection, logger)

	terratestOptions.Vars = map[string]interface{} {
		VAR_AMI_ID: amiId,
		VAR_AWS_REGION: resourceCollection.AwsRegion,
		VAR_S3_BUCKET_NAME: s3BucketName(resourceCollection),
		VAR_VAULT_CLUSTER_NAME: fmt.Sprintf("vault-test-%s", resourceCollection.UniqueId),
		VAR_CONSUL_CLUSTER_NAME: fmt.Sprintf("consul-test-%s", resourceCollection.UniqueId),
		VAR_CONSUL_CLUSTER_TAG_KEY: fmt.Sprintf("consul-test-%s", resourceCollection.UniqueId),
		VAR_SSH_KEY_NAME: resourceCollection.KeyPair.Name,
		VAR_FORCE_DESTROY_S3_BUCKET: boolToTerraformVar(true),
		VAULT_CLUSTER_PUBLIC_VAR_CREATE_DNS_ENTRY: boolToTerraformVar(false),
		VAULT_CLUSTER_PUBLIC_VAR_HOSTED_ZONE_DOMAIN_NAME: "",
		VAULT_CLUSTER_PUBLIC_VAR_VAULT_DOMAIN_NAME: "",
	}

	deploy(t, terratestOptions)
	initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terratestOptions, resourceCollection, logger)
	testVaultViaElb(t, terratestOptions, logger)
}

// Initialize the Vault cluster and unseal each of the nodes by connecting to them over SSH and executing Vault
// commands. The reason we use SSH rather than using the Vault client remotely is we want to verify that the
// self-signed TLS certificate is properly configured on each server so when you're on that server, you don't
// get errors about the certificate being signed by an unknown party.
func initializeAndUnsealVaultCluster(t *testing.T, asgNameOutputVar string, sshUserName string, terratestOptions *terratest.TerratestOptions, resourceCollection *terratest.RandomResourceCollection, logger *log.Logger) VaultCluster {
	cluster := findVaultClusterNodes(t, asgNameOutputVar, sshUserName, terratestOptions, resourceCollection)

	establishConnectionToCluster(t, cluster, logger)
	waitForVaultToBoot(t, cluster, logger)
	initializeVault(t, &cluster, logger)

	assertStatus(t, cluster.Leader, Sealed, logger)
	unsealVaultNode(t, cluster.Leader, cluster.UnsealKeys, logger)
	assertStatus(t, cluster.Leader, Leader, logger)

	assertStatus(t, cluster.Standby1, Sealed, logger)
	unsealVaultNode(t, cluster.Standby1, cluster.UnsealKeys, logger)
	assertStatus(t, cluster.Standby1, Standby, logger)

	assertStatus(t, cluster.Standby2, Sealed, logger)
	unsealVaultNode(t, cluster.Standby2, cluster.UnsealKeys, logger)
	assertStatus(t, cluster.Standby2, Standby, logger)

	return cluster
}

// Find the nodes in the given Vault ASG and return them in a VaultCluster struct
func findVaultClusterNodes(t *testing.T, asgNameOutputVar string, sshUserName string, terratestOptions *terratest.TerratestOptions, resourceCollection *terratest.RandomResourceCollection) VaultCluster {
	asgName, err := terratest.Output(terratestOptions, asgNameOutputVar)
	if err != nil {
		t.Fatalf("Could not read output %s due to error: %v", asgNameOutputVar, err)
	}

	nodeIpAddresses := getIpAddressesOfAsgInstances(t, asgName, resourceCollection.AwsRegion)
	if len(nodeIpAddresses) != 3 {
		t.Fatalf("Expected to get three IP addresses for Vault cluster, but got %d: %v", len(nodeIpAddresses), nodeIpAddresses)
	}

	return VaultCluster{
		Leader: ssh.Host{
			Hostname: nodeIpAddresses[0],
			SshUserName: sshUserName,
			SshKeyPair: resourceCollection.KeyPair,
		},

		Standby1: ssh.Host {
			Hostname: nodeIpAddresses[1],
			SshUserName: sshUserName,
			SshKeyPair: resourceCollection.KeyPair,
		},

		Standby2: ssh.Host {
			Hostname: nodeIpAddresses[2],
			SshUserName: sshUserName,
			SshKeyPair: resourceCollection.KeyPair,
		},
	}
}

// Wait until we can connect to each of the Vault cluster EC2 Instances
func establishConnectionToCluster(t *testing.T, cluster VaultCluster, logger *log.Logger) {
	for _, node := range cluster.Nodes() {
		description := fmt.Sprintf("Trying to establish SSH connection to %s", node.Hostname)
		logger.Println(description)

		maxRetries := 30
		sleepBetweenRetries := 10 * time.Second

		_, err := util.DoWithRetry(description, maxRetries, sleepBetweenRetries, logger, func() (string, error) {
			return "", ssh.CheckSshConnection(node, logger)
		})

		if err != nil {
			t.Fatalf("Failed to establish connection to host %s: %v", node.Hostname, err)
		}
	}
}

// Wait until the Vault servers are booted the very first time on the EC2 Instance. As a simple solution, we simply
// wait for the leader to boot and assume if it's up, the other nodes will be too.
func waitForVaultToBoot(t *testing.T, cluster VaultCluster, logger *log.Logger) {
	for _, node := range cluster.Nodes() {
		logger.Printf("Waiting for Vault to boot the first time on host %s. Expecting it to be in uninitialized status (%d).", node.Hostname, int(Uninitialized))
		assertStatus(t, node, Uninitialized, logger)
	}
}

// Initialize the Vault cluster, filling in the unseal keys in the given vaultCluster struct
func initializeVault(t * testing.T, vaultCluster *VaultCluster, logger *log.Logger) {
	logger.Println("Initializing the cluster")
	output, err := ssh.CheckSshCommand(vaultCluster.Leader, "vault init", logger)
	if err != nil {
		t.Fatalf("Failed to initalize Vault: %v", err)
	}

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
func s3BucketName(resourceCollection *terratest.RandomResourceCollection) string {
	return strings.ToLower(fmt.Sprintf("vault-blueprint-test-%s", resourceCollection.UniqueId))
}

// SSH to a Vault node and make sure that is properly configured to use Consul for DNS so that the vault.service.consul
// domain name works.
func testVaultUsesConsulForDns(t *testing.T, cluster VaultCluster, logger *log.Logger) {
	// Pick any host, it shouldn't matter
	host := cluster.Standby1

	command := "vault status -address=https://vault.service.consul:8200"
	description := fmt.Sprintf("Checking that the Vault server at %s is properly configured to use Consul for DNS: %s", host.Hostname, command)
	logger.Println(description)


	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	out, err := util.DoWithRetry(description, maxRetries, sleepBetweenRetries, logger, func() (string, error) {
		return ssh.CheckSshCommand(host, command, logger)
	})

	logger.Printf("Output from command vault status call to vault.service.consul: %s", out)
	if err != nil {
		t.Fatalf("Failed to run vault command with vault.service.consul URL due to error: %v", err)
	}
}

// Use the Vault client to connect to the Vault via the ELB, via the public DNS entry, and make sure it works without
// Vault or TLS errors
func testVaultViaElb(t *testing.T, terratestOptions *terratest.TerratestOptions, logger *log.Logger) {
	domainName := getElbDomainName(t, terratestOptions)
	description := fmt.Sprintf("Testing Vault via ELB at domain name %s", domainName)
	logger.Printf(description)

	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	vaultClient := createVaultClient(t, domainName)

	out, err := util.DoWithRetry(description, maxRetries, sleepBetweenRetries, logger, func() (string, error) {
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

	if err != nil {
		t.Fatalf("Failed to use Vault client with ELB to validate that the cluster is initialized: %v", err)
	}

	logger.Printf(out)
}

// Get the ELB domain name
func getElbDomainName(t *testing.T, terratestOptions *terratest.TerratestOptions) string {
	domainName, err := terratest.Output(terratestOptions, VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME)
	if err != nil {
		t.Fatalf("Failed to read output %s: %v", VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME, err)
	}
	if domainName == "" {
		t.Fatalf("Domain name output %s was empty", VAULT_CLUSTER_PUBLIC_OUTPUT_ELB_DNS_NAME)
	}
	return domainName
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
func unsealVaultNode(t *testing.T, host ssh.Host, unsealKeys []string, logger *log.Logger) {
	unsealCommands := []string{}
	for _, unsealKey := range unsealKeys {
		unsealCommands = append(unsealCommands, fmt.Sprintf("vault unseal %s", unsealKey))
	}

	unsealCommand := strings.Join(unsealCommands, " && ")

	logger.Printf("Unsealing Vault on host %s", host.Hostname)
	_, err := ssh.CheckSshCommand(host, unsealCommand, logger)
	if err != nil {
		t.Fatalf("Failed to unseal cluster due to error: %v", err)
	}
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
func assertStatus(t *testing.T, host ssh.Host, expectedStatus VaultStatus, logger *log.Logger) {
	description := fmt.Sprintf("Check that the Vault node %s has status %d", host.Hostname, int(expectedStatus))
	logger.Println(description)

	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	out, err := util.DoWithRetry(description, maxRetries, sleepBetweenRetries, logger, func() (string, error) {
		return checkStatus(host, expectedStatus, logger)
	})

	if err != nil {
		t.Fatal(err)
	}

	logger.Printf(out)
}

// Delete the temporary self-signed cert files we created
func cleanupTlsCertFiles(tlsCert TlsCert) {
	os.Remove(tlsCert.CAPublicKeyPath)
	os.Remove(tlsCert.PrivateKeyPath)
	os.Remove(tlsCert.PublicKeyPath)
}

// Check the status of the given Vault node and ensure it matches the expected status. Note that we use curl to do the
// status check so we can ensure that TLS certificates work for curl (and not just the Vault client).
func checkStatus(host ssh.Host, expectedStatus VaultStatus, logger *log.Logger) (string, error) {
	curlCommand := "curl -s -o /dev/null -w '%{http_code}' https://127.0.0.1:8200/v1/sys/health"
	logger.Printf("Using curl to check status of Vault server %s: %s", host.Hostname, curlCommand)

	output, err := ssh.CheckSshCommand(host, curlCommand, logger)
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