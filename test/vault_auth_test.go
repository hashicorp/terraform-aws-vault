package test

import (
	"testing"
)

func TestVaultEC2AuthWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultEC2AuthTest(t, "ubuntu16-ami")
}

func TestVaultEC2AuthWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultEC2AuthTest(t, "amazon-linux-ami")
}

func TestVaultIAMAuthWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultIAMAuthTest(t, "ubuntu16-ami")
}

func TestVaultIAMAuthWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultIAMAuthTest(t, "amazon-linux-ami")
}
