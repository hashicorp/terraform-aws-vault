package test

import (
	"testing"
)

func TestVaultAuthWithUbuntuAmi(t *testing.T) {
	t.Parallel()
	runVaultEC2AuthTest(t, "ubuntu16-ami")
}

func TestVaultAuthWithAmazonLinuxAmi(t *testing.T) {
	t.Parallel()
	runVaultEC2AuthTest(t, "amazon-linux-ami")
}
