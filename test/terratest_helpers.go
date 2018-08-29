package test

import (
	"testing"

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
func buildAmi(t *testing.T, packerTemplatePath string, packerBuildName string, tlsCert TlsCert, awsRegion string) string {
	options := &packer.Options{
		Template: packerTemplatePath,
		Only:     packerBuildName,
		Vars: map[string]string{
			AMI_VAR_AWS_REGION:      awsRegion,
			AMI_VAR_CA_PUBLIC_KEY:   tlsCert.CAPublicKeyPath,
			AMI_VAR_TLS_PUBLIC_KEY:  tlsCert.PublicKeyPath,
			AMI_VAR_TLS_PRIVATE_KEY: tlsCert.PrivateKeyPath,
		},
	}

	return packer.BuildAmi(t, options)
}

// Use Packer to build the AMI in the given packer template, with the given build name, and return the AMI's ID
func buildAmiWithDownloadEnv(t *testing.T, packerTemplatePath string, packerBuildName string, tlsCert TlsCert, awsRegion string, vaultDownloadUrl string) string {
	options := &packer.Options{
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

	return packer.BuildAmi(t, options)
}

func saveTlsCert(t *testing.T, testFolder string, tlsCert TlsCert) {
	test_structure.SaveTestData(t, test_structure.FormatTestDataPath(testFolder, SAVED_TLS_CERT), tlsCert)
}

func loadTlsCert(t *testing.T, testFolder string) TlsCert {
	var tlsCert TlsCert
	test_structure.LoadTestData(t, test_structure.FormatTestDataPath(testFolder, SAVED_TLS_CERT), &tlsCert)
	return tlsCert
}
