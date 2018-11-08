package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/require"
)

const VAULT_CLUSTER_S3_BACKEND_PATH = "examples/vault-s3-backend"

const VAR_ENABLE_S3_BACKEND = "enable_s3_backend"
const VAR_S3_BUCKET_NAME = "s3_bucket_name"
const VAR_FORCE_DESTROY_S3_BUCKET = "force_destroy_s3_bucket"

// Test the Vault with S3 Backend example by:
//
// 1. Copy the code in this repo to a temp folder so tests on the Terraform code can run in parallel without the
//    state files overwriting each other.
// 2. Build the AMI in the vault-consul-ami example with the given build name
// 3. Deploy that AMI using the example Terraform code
// 4. SSH to a Vault node and initialize the Vault cluster
// 5. SSH to each Vault node and unseal it
// 6. Connect to the Vault cluster via the ELB
func runVaultWithS3BackendClusterTest(t *testing.T, amiId string, sshUserName string) {
	examplesDir := test_structure.CopyTerraformFolderToTemp(t, REPO_ROOT, VAULT_CLUSTER_S3_BACKEND_PATH)

	defer test_structure.RunTestStage(t, "teardown", func() {
		teardownResources(t, examplesDir)
	})

	defer test_structure.RunTestStage(t, "logs", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, WORK_DIR, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)
		asgName := terraform.OutputRequired(t, terraformOptions, OUTPUT_VAULT_CLUSTER_ASG_NAME)

		sysLogPath := vaultSyslogPathUbuntu
		if sshUserName == "ec2-user" {
			sysLogPath = vaultSyslogPathAmazonLinux
		}

		instanceIdToFilePathToContents := aws.FetchContentsOfFilesFromAsg(t, awsRegion, sshUserName, keyPair, asgName, true, vaultStdOutLogFilePath, vaultStdErrLogFilePath, sysLogPath)

		require.Len(t, instanceIdToFilePathToContents, vaultClusterSizeInExamples)

		for instanceID, filePathToContents := range instanceIdToFilePathToContents {
			require.Contains(t, filePathToContents, vaultStdOutLogFilePath)
			require.Contains(t, filePathToContents, vaultStdErrLogFilePath)
			require.Contains(t, filePathToContents, sysLogPath)

			logger.Logf(t, "Contents of %s on Instance %s:\n\n%s\n", vaultStdOutLogFilePath, instanceID, filePathToContents[vaultStdOutLogFilePath])
			logger.Logf(t, "Contents of %s on Instance %s:\n\n%s\n", vaultStdErrLogFilePath, instanceID, filePathToContents[vaultStdErrLogFilePath])
			logger.Logf(t, "Contents of %s on Instance %s:\n\n%s\n", sysLogPath, instanceID, filePathToContents[sysLogPath])
		}
	})

	test_structure.RunTestStage(t, "deploy", func() {
		uniqueId := random.UniqueId()
		terraformVars := map[string]interface{}{
			VAR_ENABLE_S3_BACKEND:       boolToTerraformVar(true),
			VAR_S3_BUCKET_NAME:          s3BucketName(uniqueId),
			VAR_FORCE_DESTROY_S3_BUCKET: boolToTerraformVar(true),
		}
		deployCluster(t, amiId, examplesDir, uniqueId, terraformVars)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, examplesDir)
		awsRegion := test_structure.LoadString(t, WORK_DIR, SAVED_AWS_REGION)
		keyPair := test_structure.LoadEc2KeyPair(t, examplesDir)

		cluster := initializeAndUnsealVaultCluster(t, OUTPUT_VAULT_CLUSTER_ASG_NAME, sshUserName, terraformOptions, awsRegion, keyPair)
		testVaultUsesConsulForDns(t, cluster)
	})
}
