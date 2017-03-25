package test

import (
	"testing"
	"path/filepath"
	"github.com/gruntwork-io/terratest"
	"os"
	"os/user"
	"io/ioutil"
)

type TlsCert struct {
	CAPublicKeyPath  string
	PublicKeyPath    string
	PrivateKeyPath   string
}

const PRIVATE_TLS_CERT_PATH = "modules/private-tls-cert"

const VAR_CA_PUBLIC_KEY_FILE_PATH = "ca_public_key_file_path"
const VAR_PUBLIC_KEY_FILE_PATH = "public_key_file_path"
const VAR_PRIVATE_KEY_FILE_PATH = "private_key_file_path"
const VAR_OWNER = "owner"
const VAR_ORGANIZATION_NAME = "organization_name"
const VAR_CA_COMMON_NAME = "ca_common_name"
const VAR_COMMON_NAME = "common_name"
const VAR_DNS_NAMES = "dns_names"
const VAR_IP_ADDRESSES = "ip_addresses"
const VAR_VALIDITY_PERIOD_HOURS = "validity_period_hours"

// Use the private-tls-cert module to generate a self-signed TLS certificate
func generateSelfSignedTlsCert(t *testing.T, testName string) TlsCert {
	rootTempPath := copyRepoToTempFolder(t, REPO_ROOT)
	defer os.RemoveAll(rootTempPath)

	resourceCollection := createBaseRandomResourceCollection(t)
	terratestOptions := createBaseTerratestOptions(t, testName, filepath.Join(rootTempPath, PRIVATE_TLS_CERT_PATH), resourceCollection)
	defer terratest.Destroy(terratestOptions, resourceCollection)

	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Couldn't get current OS user: %v", err)
	}

	caPublicKeyFilePath, err := ioutil.TempFile("", "ca-public-key")
	if err != nil {
		t.Fatalf("Couldn't create temp file: %v", err)
	}

	publicKeyFilePath, err := ioutil.TempFile("", "tls-public-key")
	if err != nil {
		t.Fatalf("Couldn't create temp file: %v", err)
	}

	privateKeyFilePath, err := ioutil.TempFile("", "tls-private-key")
	if err != nil {
		t.Fatalf("Couldn't create temp file: %v", err)
	}

	terratestOptions.Vars = map[string]interface{}{
		VAR_CA_PUBLIC_KEY_FILE_PATH: caPublicKeyFilePath.Name(),
		VAR_PUBLIC_KEY_FILE_PATH: publicKeyFilePath.Name(),
		VAR_PRIVATE_KEY_FILE_PATH: privateKeyFilePath.Name(),
		VAR_OWNER: currentUser.Username,
		VAR_ORGANIZATION_NAME: "Gruntwork",
		VAR_CA_COMMON_NAME: "Vault Blueprint Test CA",
		VAR_COMMON_NAME: "Vault Blueprint Test",
		VAR_DNS_NAMES: []string{"vault.service.consul"},
		VAR_IP_ADDRESSES: []string{"127.0.0.1"},
		VAR_VALIDITY_PERIOD_HOURS: 1000,
	}

	if _, err := terratest.Apply(terratestOptions); err != nil {
		t.Fatalf("Failed to create TLS certs: %v", err)
	}

	return TlsCert{
		CAPublicKeyPath: caPublicKeyFilePath.Name(),
		PublicKeyPath: publicKeyFilePath.Name(),
		PrivateKeyPath: privateKeyFilePath.Name(),
	}
}
