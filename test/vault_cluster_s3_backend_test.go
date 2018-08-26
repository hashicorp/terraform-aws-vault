package test

import "testing"

func TestVaultClusterS3BackendWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping this test as it is failing intermittently.") // TODO fix this test!!!
	runVaultWithS3BackendClusterTest(t, "ubuntu16-ami", "ubuntu")
}

func TestVaultClusterS3BackendAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping this test as it is failing intermittently.") // TODO fix this test!!!
	runVaultWithS3BackendClusterTest(t, "amazon-linux-ami", "ec2-user")
}
