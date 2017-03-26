package test

import (
	"testing"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Get the public IP addresses of the EC2 Instances in an Auto Scaling Group of the given name in the given
// region
func getIpAddressesOfAsgInstances(t *testing.T, asgName string, awsRegion string) []string {
	instanceIds := getIdsOfAsgInstances(t, asgName, awsRegion)
	return getPublicIpsOfEc2Instances(t, instanceIds, awsRegion)
}

// Get the instance IDs of the EC2 Instances in an Auto Scaling Group of the given name in the given region
func getIdsOfAsgInstances(t *testing.T, asgName string, awsRegion string) []string {
	autoscalingClient := createAutoscalingClient(t, awsRegion)

	input := autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{aws.String(asgName)}}
	output, err := autoscalingClient.DescribeAutoScalingGroups(&input)
	if err != nil {
		t.Fatalf("Failed to call DescribeAutoScalingGroupsInput API due to error: %v", err)
	}

	ids := []string{}
	for _, asg := range output.AutoScalingGroups {
		for _, instance := range asg.Instances {
			ids = append(ids, *instance.InstanceId)
		}
	}

	if len(ids) == 0 {
		t.Fatalf("Failed to find any instance IDs for asg %s", asgName)
	}

	return ids
}

// Get the public IP addresses of the given EC2 Instances in the given region
func getPublicIpsOfEc2Instances(t *testing.T, instanceIds []string, awsRegion string) []string {
	ec2Client := createEc2Client(t, awsRegion)

	input := ec2.DescribeInstancesInput{InstanceIds: aws.StringSlice(instanceIds)}
	output, err := ec2Client.DescribeInstances(&input)
	if err != nil {
		t.Fatalf("Failed to fetch information about EC2 Instances %v due to error: %v", instanceIds, err)
	}

	ipAddresses := []string{}
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			ipAddresses = append(ipAddresses, *instance.PublicIpAddress)
		}
	}

	if len(ipAddresses) == 0 {
		t.Fatalf("Failed to find the public IP addresses for instances %v", instanceIds)
	}

	return ipAddresses
}

// Create a client that can be used to make EC2 API calls
func createEc2Client(t *testing.T, awsRegion string) *ec2.EC2 {
	awsConfig := createAwsConfig(t, awsRegion)
	return ec2.New(session.New(), awsConfig)
}

// Create a client that can be used to make Auto Scaling API calls
func createAutoscalingClient(t *testing.T, awsRegion string) *autoscaling.AutoScaling {
	awsConfig := createAwsConfig(t, awsRegion)
	return autoscaling.New(session.New(), awsConfig)
}

// Create an AWS config. This method will check for credentials and fail the test if it can't find them.
func createAwsConfig(t *testing.T, awsRegion string) *aws.Config {
	config := defaults.Get().Config.WithRegion(awsRegion)

	_, err := config.Credentials.Get()
	if err != nil {
		t.Fatalf("Error finding AWS credentials (did you set the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables?). Underlying error: %v", err)
	}

	return config
}
