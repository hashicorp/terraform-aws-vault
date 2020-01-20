# Vault auto unseal example

This folder shows an example of Terraform code that deploys a [Vault][vault] cluster
in AWS with [auto unseal][auto_unseal]. Auto unseal is a Vault feature
that automatically [unseals][seal] each node in the cluster at boot using [Amazon KMS][kms].
Without auto unseal, Vault operators are expected to manually unseal each Vault node
after it boots, a cumbersome process that typically requires multiple Vault operators
to each enter a Vault master key shard.

This example creates a private Vault cluster that is accessible only from within
the VPC within the AWS account in which it resides, or other VPCs that are peered
with the Vault VPC. The Vault cluster uses [Consul][consul] as the storage backend,
so this example also deploys a separate Consul server cluster using the
[consul-cluster module][consul_cluster] from the Consul AWS Module. Each of the
servers in this example has [Dnsmasq][dnsmasq] installed (via the [install-dnsmasq module][dnsmasq_module])
or [setup-systemd-resolved][setup_systemd_resolved] (in the case of Ubuntu 18.04)
which allows them to use the Consul server cluster for service discovery and thereby
access Vault via DNS using the domain name `vault.service.consul`.

For more info on how the Vault cluster works, check out the [vault-cluster][vault_cluster]
documentation.

**Note**: To keep this example as simple to deploy and test as possible, it deploys
the Vault cluster into your default VPC and default subnets, all of which are publicly
accessible. This is OK for learning and experimenting, but for production usage,
we strongly recommend deploying the Vault cluster into the private subnets of a custom VPC.

**Billing Warning**: Every time you create a KMS key, you're charged $1 for the month,
even if you immediately delete it.


### Quick start

1. `git clone` this repo to your computer.
1. Build a Vault and Consul AMI. See the [vault-consul-ami example][vault_consul_ami]
  documentation for instructions. Don't forget to set the variable `vault_download_url`
  with the url of the enterprise version of Vault if you wish to use Vault Enterprise.
  Make sure to note down the ID of the AMI.
1. Install [Terraform][terraform].
1. [Create an AWS KMS key][key_creation]. Take note of the key alias.
1. Open `variables.tf`, set the environment variables specified at the top of the file,
  and fill in any other variables that don't have a default. Put the AMI ID you
  previously took note into the `ami_id` variable and the KMS key alias into
  `auto_unseal_kms_key_alias`.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script][examples_helper] to
   print out the IP addresses of the Vault server and some example commands you
   can run to interact with the cluster: `../vault-examples-helper/vault-examples-helper.sh`.
1. Ssh to an instance in the vault cluster and run `vault operator init` to initialize
  the cluster, then `vault status` to check that it is unsealed. If you ssh to a
  different node in the cluster, you might have to restart Vault first with
  `sudo systemctl restart vault.service` so it will rejoin the cluster and unseal.
  To avoid doing that, you can start your cluster with initially just one node and
  start the server, then change the `vault_cluster_size` variable back to 3 and and
  run `terraform apply again`. The new nodes will join the cluster already unsealed
  in this case.

### Seal

All data stored by Vault is encrypted with a Master Key which is not stored anywhere
and Vault only ever keeps in memory. When Vault first boots, it does not have the
Master Key in memory, and therefore it can access its storage, but it cannot decrypt
its own data. So you can't really do anything apart from unsealing it or checking
the server status. While Vault is at this state, we say it is "sealed".

Since vault uses [Shamir's Secret Sharing][shamir], which splits the master key into
pieces, running `vault operator unseal <unseal key>` adds piece by piece until there
are enough parts to reconstruct the master key. This is done on different machines in the
vault cluster for better security. When Vault is unsealed and it has the recreated
master key in memory, it can then be used to read the stored decryption keys, which
can decrypt the data, and then you can start performing other operations on Vault.
Vault remains unsealed until it reboots or until someone manually reseals it.

### Auto-unseal

Vault has a feature that allows automatic unsealing via Amazon KMS. It
allows operators to delegate the unsealing process to AWS, which is useful for failure
situations where the server has to restart and then it will be already unsealed or
for the creation of ephemeral clusters. This process uses an AWS KMS key as
a [seal wrap][seal_wrap] mechanism: it encrypts and decrypts Vault's master key
(and it does so with the whole key, replacing the Shamir's Secret Sharing method).

This feature is enabled by adding a `awskms` stanza at Vault's configuration. This
module takes this into consideration on the [`run-vault`][run_vault] binary, allowing
you to pass the following flags to it:
 * `--enable-auto-unseal`: Enables the AWS KMS Auto-unseal feature and adds the `awskms`
 stanza to the configuration
 * `--auto-unseal-kms-key-id`: The key id of the AWS KMS key to be used
 * `--auto-unseal-region`: The AWS region where the KMS key lives

In this example, like in other examples, we execute `run-vault` at the [`user-data`
script][user_data], which runs on boot for every node in the Vault cluster. The
`key-id` is passed to this script by Terraform, after Terraform reads this value from a
data source through the key alias. This means that the AWS key has to be previously
manually created and we are using Terraform just to find this resource, not to
create it. It is important to notice that AWS KMS keys have a [cost][kms_pricing]
per month per key, as well as an API usage cost.

```
data "aws_kms_alias" "vault-example" {
  name = "alias/${var.auto_unseal_kms_key_alias}"
}
```

If you wish to use Vault Enterprise, you still need to apply your Vault
Enterprise License to the cluster with `vault write /sys/license "text=$LICENSE_KEY_TEXT"`.

[ami]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html
[auto_unseal]: https://www.vaultproject.io/docs/enterprise/auto-unseal/index.html
[consul_cluster]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/consul-cluster
[consul]: https://www.consul.io/
[dnsmasq_module]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[setup_systemd_resolved]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/setup-systemd-resolved
[examples_helper]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh
[key_creation]: https://docs.aws.amazon.com/kms/latest/developerguide/create-keys.html
[kms]: https://aws.amazon.com/kms/
[kms_pricing]: https://aws.amazon.com/kms/pricing/
[run_vault]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/run-vault
[seal_wrap]: https://www.vaultproject.io/docs/enterprise/sealwrap/index.html
[seal]: https://www.vaultproject.io/docs/concepts/seal.html
[shamir]: https://en.wikipedia.org/wiki/Shamir%27s_Secret_Sharing
[terraform]: https://www.terraform.io/
[user_data]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-auto-unseal/user-data-vault.sh
[vault_cluster]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster
[vault_consul_ami]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami
[vault]: https://www.vaultproject.io/
