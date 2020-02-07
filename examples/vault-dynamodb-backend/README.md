# Vault Cluster with DynamoDB backend example

This folder shows an example of Terraform code to deploy a [Vault](https://www.vaultproject.io/) cluster in
[AWS](https://aws.amazon.com/) using the [vault-cluster module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster).
The Vault cluster uses [DynamoDB](https://aws.amazon.com/dynamodb/) as a high-availability storage backend and [S3](https://aws.amazon.com/s3/)
for durable storage, so this example also deploys a separate DynamoDB table

This example creates a Vault cluster spread across the subnets in the default VPC of the AWS account. For an example of a Vault cluster
that is publicly accessible, see [the root example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/root-example).

![Vault architecture](https://github.com/hashicorp/terraform-aws-vault/blob/master/_docs/architecture-with-dynamodb.png?raw=true)

You will need to create an [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html)
that has Vault installed, which you can do using the [vault-consul-ami example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami)).  

For more info on how the Vault cluster works, check out the [vault-cluster](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster) documentation.

**Note**: To keep this example as simple to deploy and test as possible, it deploys the Vault cluster into your default
VPC and default subnets, some of which might be publicly accessible. This is OK for learning and experimenting, but for
production usage, we strongly recommend deploying the Vault cluster into the private subnets of a custom VPC.




## Quick start

To deploy a Vault Cluster:

1. `git clone` this repo to your computer.
1. Optional: build a Vault and Consul AMI. See the [vault-consul-ami
   example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami) documentation for
   instructions. Make sure to note down the ID of the AMI.
1. Install [Terraform](https://www.terraform.io/).
1. Open `variables.tf`, set the environment variables specified at the top of the file, and fill in any other variables that
   don't have a default. If you built a custom AMI, put the AMI ID into the `ami_id` variable. Otherwise, one of our
   public example AMIs will be used by default. These AMIs are great for learning/experimenting, but are NOT
   recommended for production use.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh) to
   print out the IP addresses of the Vault servers and some example commands you can run to interact with the cluster:
   `../vault-examples-helper/vault-examples-helper.sh`.

To see how to connect to the Vault cluster, initialize it, and start reading and writing secrets, head over to the
[How do you use the Vault cluster?](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster#how-do-you-use-the-vault-cluster) docs.
