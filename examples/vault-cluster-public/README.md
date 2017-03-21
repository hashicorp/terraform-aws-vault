# Public Vault Cluster Example 

This folder shows an example of Terraform code to deploy a [Vault](https://www.vaultproject.io/) cluster in 
[AWS](https://aws.amazon.com/) using the [vault-cluster](/modules/vault-cluster) and [vault-elb](/modules/vault-elb) 
modules. The Vault cluster uses [Consul](https://www.consul.io/) as a storage backend, so this example also deploys a 
separate Consul cluster using the [consul-cluster 
module](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/consul-cluster) from the Consul AWS 
Blueprint.

This example creates a public Vault cluster that is accessible from the public Internet via an [Elastic Load Balancer 
(ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/). For an example of a private Vault cluster
that is accessible from inside the AWS account, see [vault-cluster-private](/examples/vault-cluster-private).

![Vault architecture](/_docs/architecture-elb.png)

You will need to create an [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) 
that has Vault and Consul installed, which you can do using the [vault-consul-ami example](/examples/vault-consul-ami)).  

For more info on how the Vault cluster works, check out the [vault-cluster](/modules/vault-cluster) documentation.




## Quick start

To deploy a Vault Cluster:

1. `git clone` this repo to your computer.
1. Build a Vault and Consul AMI. See the [vault-consul-ami example](/examples/vault-consul-ami) documentation for 
   instructions. Make sure to note down the ID of the AMI.
1. Install [Terraform](https://www.terraform.io/).
1. Open `vars.tf`, set the environment variables specified at the top of the file, and fill in any other variables that
   don't have a default, including putting your AMI ID into the `ami_id` variable.
1. Run `terraform get`.
1. Run `terraform plan`.
1. If the plan looks good, run `terraform apply`.

After the `apply` command finishes, a Vault and Consul cluster will boot up and discover each other.
 
To see how to connect to the Vault cluster and start reading and writing secrets, head over to the [How do you connect 
to the Vault cluster?](/modules/vault-cluster#how-do-you-connect-to-the-vault-cluster) docs.
