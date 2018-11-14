package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/gruntwork-io/terratest/modules/test-structure"
)

const AMI_VAR_AWS_REGION = "aws_region"
const AMI_VAR_CA_PUBLIC_KEY = "ca_public_key_path"
const AMI_VAR_TLS_PUBLIC_KEY = "tls_public_key_path"
const AMI_VAR_TLS_PRIVATE_KEY = "tls_private_key_path"
const AMI_VAR_VAULT_DOWNLOAD_URL = "VAULT_DOWNLOAD_URL"

const SAVED_TLS_CERT = "TlsCert"

// Use Packer to build the AMI in the given packer template, with the given build name, and return the AMI's ID
func composeAmiOptions(t *testing.T, packerTemplatePath string, packerBuildName string, tlsCert TlsCert, awsRegion string, vaultDownloadUrl string) *packer.Options {
	return &packer.Options{
		Template: packerTemplatePath,
		Only:     packerBuildName,
		Vars: map[string]string{
			AMI_VAR_AWS_REGION:      awsRegion,
			AMI_VAR_CA_PUBLIC_KEY:   tlsCert.CAPublicKeyPath,
			AMI_VAR_TLS_PUBLIC_KEY:  tlsCert.PublicKeyPath,
			AMI_VAR_TLS_PRIVATE_KEY: tlsCert.PrivateKeyPath,
		},
		Env: map[string]string{
			AMI_VAR_VAULT_DOWNLOAD_URL: vaultDownloadUrl,
		},
	}
}

func saveTlsCert(t *testing.T, testFolder string, tlsCert TlsCert) {
	test_structure.SaveTestData(t, test_structure.FormatTestDataPath(testFolder, SAVED_TLS_CERT), tlsCert)
}

func loadTlsCert(t *testing.T, testFolder string) TlsCert {
	var tlsCert TlsCert
	test_structure.LoadTestData(t, test_structure.FormatTestDataPath(testFolder, SAVED_TLS_CERT), &tlsCert)
	return tlsCert
}

func writeLogFile(t *testing.T, buffer string, destination string) {
	file, err := os.Create(destination)
	if err != nil {
		logger.Logf(t, fmt.Sprintf("Error creating log file on disk: %s", err.Error()))
	}
	defer file.Close()

	file.WriteString(buffer)
}
