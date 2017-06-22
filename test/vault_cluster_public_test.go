package test

import "testing"

func TestVaultClusterPublicWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultPublicClusterTest(t, "vault-public-ubuntu", "ubuntu16-ami", "ubuntu")
}

func TestVaultClusterPublicAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultPublicClusterTest(t, "vault-public-amzn", "amazon-linux-ami", "ec2-user")
}

