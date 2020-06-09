# Private Vault Cluster Example

This folder shows an example of Terraform code to deploy a [Vault](https://www.vaultproject.io/) cluster in
[AWS](https://aws.amazon.com/) using the [vault-cluster module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster). The Vault cluster uses
[Consul](https://www.consul.io/) as a storage backend, so this example also deploys a separate Consul server cluster
using the [consul-cluster module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/consul-cluster)
from the Consul AWS Module.

This example creates a private Vault cluster, which is private in the sense that the EC2 Instances are not fronted by a
load balancer, as is the case in the [Vault Public Example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/root-example). Keep in mind that if the Vault
nodes are deployed to public subnets (i.e. subnets that have a route to the public Internet), this "private" cluster will
still be accessible from the public Internet.

Each of the servers in this example has [Dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html) installed (via the
[install-dnsmasq module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq)) or
[setup-systemd-resolved](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/setup-systemd-resolved)
(in the case Ubuntu of 18.04) 
which allows it to use the Consul server cluster for service discovery and thereby access Vault via DNS using the
domain name `vault.service.consul`. For an example of a Vault cluster
that is publicly accessible, see [the root example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/root-example).

![Vault architecture](https://github.com/hashicorp/terraform-aws-vault/blob/master/_docs/architecture.png?raw=true)

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
