# Vault auto unseal example

** For Vault Enterprise version only **

This folder shows an example of Terraform code to deploy a [Vault][vault] cluster in
Amazon AWS using the [vault-cluster module][vault_cluster] and use the Vault Enterprise
feature of auto unsealing the cluster through Amazon KMS.


This example creates a private Vault cluster that is only accessible from servers
within the AWS account. The Vault cluster uses [Consul][consul] as a storage backend,
so this example also deploys a separate Consul server cluster using the
[consul-cluster module][consul_cluster] from the Consul AWS Module. Each of the
servers in this example has [Dnsmasq][dnsmasq] installed (via the [install-dnsmasq module][dnsmasq_module])
which allows it to use the Consul server cluster for service discovery and thereby
access Vault via DNS using the domain name `vault.service.consul`.

For more info on how the Vault cluster works, check out the [vault-cluster][vault_cluster]
documentation.

**Note**: To keep this example as simple to deploy and test as possible, it deploys
the Vault cluster into your defaultVPC and default subnets, all of which are publicly
accessible. This is OK for learning and experimenting, but for production usage,
we strongly recommend deploying the Vault cluster into the private subnets of a custom VPC.


### Quick start

1. `git clone` this repo to your computer.
1. Build a Vault and Consul AMI. See the [vault-consul-ami example][vault_consul_ami]
  documentation for instructions. Don't forget to set the variable `vault_download_url`
  with the url of the enterprise version of Vault. Make sure to note down the ID of the AMI.
1. Install [Terraform][terraform].
1. Open `vars.tf`, set the environment variables specified at the top of the file,
  and fill in any other variables that don't have a default. Put the AMI ID you
  previously took note into the `ami_id` variable.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script][examples_helper] to
   print out the IP addresses of the Vault server and some example commands you
   can run to interact with the cluster: `../vault-examples-helper/vault-examples-helper.sh`.
1. Ssh to an instance in the vault cluster and run `vault operator init` to initialize
  the cluster, then `vault status` to check that it is unsealed. (If you ssh to a
  different node in the cluster, you might have to restart Vault first with
  `sudo supervisorctl restart vault`)

[ami]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html
[consul_cluster]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/consul-cluster
[consul]: https://www.consul.io/
[dnsmasq_module]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[examples_helper]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh
[terraform]: https://www.terraform.io/
[vault_cluster]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster
[vault_consul_ami]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami
[vault]: https://www.vaultproject.io/
