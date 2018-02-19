package test

import "testing"

func TestVaultClusterS3BackendWithUbuntuAmi(t *testing.T) {
    t.Parallel()
    runVaultWithS3BackendClusterTest(t, "vault-public-s3-ubuntu", "ubuntu16-ami", "ubuntu")
}

func TestVaultClusterS3BackendAmazonLinuxAmi(t *testing.T) {
    t.Parallel()
    runVaultWithS3BackendClusterTest(t, "vault-public-s3-amzn", "amazon-linux-ami", "ec2-user")
}
