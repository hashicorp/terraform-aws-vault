package test

import (
	"testing"
)

func TestVaultClusterPrivateWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultPrivateClusterTest(t, "ubuntu16-ami", "ubuntu")
}

func TestVaultClusterPrivateWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultPrivateClusterTest(t, "amazon-linux-ami", "ec2-user")
}
