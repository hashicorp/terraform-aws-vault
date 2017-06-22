package test

import (
	"testing"
)

func TestVaultClusterPrivateWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultPrivateClusterTest(t, "vault-private-ubuntu", "ubuntu16-ami", "ubuntu")
}

func TestVaultClusterPrivateWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultPrivateClusterTest(t, "vault-private-amzn", "amazon-linux-ami", "ec2-user")
}
