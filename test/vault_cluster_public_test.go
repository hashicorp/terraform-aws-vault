package test

import "testing"

func TestVaultClusterPublicWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultPublicClusterTest(t, "ubuntu16-ami", "ubuntu")
}

func TestVaultClusterPublicAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultPublicClusterTest(t, "amazon-linux-ami", "ec2-user")
}

