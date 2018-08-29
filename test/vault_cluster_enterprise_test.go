package test

import (
	"os"
	"testing"
)

func TestVaultClusterEnterpriseWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultEnterpriseClusterTest(t, "ubuntu16-ami", "ubuntu", os.Getenv("VAULT_AMI_TEMPLATE_VAR_DOWNLOAD_URL"))
}

func TestVaultClusterEnterpriseWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultEnterpriseClusterTest(t, "amazon-linux-ami", "ec2-user", os.Getenv("VAULT_AMI_TEMPLATE_VAR_DOWNLOAD_URL"))
}
