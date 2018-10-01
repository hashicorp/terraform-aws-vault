package test

import (
	"os"
	"testing"
)

func TestVaultClusterEnterpriseWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultEnterpriseClusterTest(t, "ubuntu16-ami", "ubuntu", getUrlFromEnv(t))
}

func TestVaultClusterEnterpriseWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultEnterpriseClusterTest(t, "amazon-linux-ami", "ec2-user", getUrlFromEnv(t))
}

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
