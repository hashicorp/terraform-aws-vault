# Public Vault Cluster Example

This folder shows an example of Terraform code to deploy a [Vault](https://www.vaultproject.io/) cluster in
[AWS](https://aws.amazon.com/) using the [vault-cluster](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster) and [vault-elb](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-elb)
modules. The Vault cluster uses [Consul](https://www.consul.io/) as a storage backend, so this example also deploys a
separate Consul server cluster using the [consul-cluster
module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/consul-cluster) from the Consul AWS
Module.

This example creates a public Vault cluster that is accessible from the public Internet via an [Elastic Load Balancer
(ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/). For an example of a private Vault cluster
that is accessible from inside the AWS account, see [vault-cluster-private](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private).

![Vault architecture](https://github.com/hashicorp/terraform-aws-vault/blob/master/_docs/architecture-elb.png?raw=true)

You will need to create an [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html)
that has Vault and Consul installed, which you can do using the [vault-consul-ami example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami)).

For more info on how the Vault cluster works, check out the [vault-cluster](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster) documentation.

**Note**: To keep this example as simple to deploy and test as possible, it deploys the Vault cluster into your default
VPC and default subnets, all of which are publicly accessible. This is OK for learning and experimenting, but for
production usage, we strongly recommend deploying the Vault cluster into the private subnets of a custom VPC.




## Quick start

To deploy a Vault Cluster:

1. `git clone` this repo to your computer.
1. Optional: build a Vault and Consul AMI. See the [vault-consul-ami
   example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami) documentation for
   instructions. Make sure to note down the ID of the AMI.
1. Install [Terraform](https://www.terraform.io/).
1. Open `variables.tf`, set the environment variables specified at the top of the file, and fill in any other variables
   that don't have a default. If you built a custom AMI, put the AMI ID into the `ami_id` variable. Otherwise, one of
   our public example AMIs will be used by default. These AMIs are great for learning/experimenting, but are NOT
   recommended for production use.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh) to
   print out the IP addresses of the Vault servers and some example commands you can run to interact with the cluster:
   `../vault-examples-helper/vault-examples-helper.sh`. **NOTE**: This script assumes that you have a valid SSH key set
   for the variable `ssh_key_name`.

To see how to connect to the Vault cluster, initialize it, and start reading and writing secrets, head over to the
[How do you use the Vault cluster?](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster#how-do-you-use-the-vault-cluster) docs.
