package test

import (
	"github.com/gruntwork-io/terratest/modules/aws"
	"testing"
)

// Get the public IP addresses of the EC2 Instances in an Auto Scaling Group of the given name in the given
// region
func getIpAddressesOfAsgInstances(t *testing.T, asgName string, awsRegion string) []string {
	instanceIds := aws.GetInstanceIdsForAsg(t, asgName, awsRegion)
	instanceIdsToIps := aws.GetPublicIpsOfEc2Instances(t, instanceIds, awsRegion)

	ips := []string{}
	for _, ip := range instanceIdsToIps {
		ips = append(ips, ip)
	}

	return ips
}
