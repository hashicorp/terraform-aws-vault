# Vault agent example

This example shows how to use Vault agent's [auto-auth][auto_auth] feature to authenticate
to a [vault cluster][vault_cluster].  Vault agent automatically handles renewal and re-authentication
and thus you do not have to implement potentially complicated renewal logic yourself.

This example uses the [AWS IAM Auth Method][iam_auth] to authenticate, and builds upon the [`IAM` auth
example][iam_example], creating the same Vault `example-role`.  The difference between that and this
example is instead of using curl to access the Vault API to authenticate, this example uses
Vault agent to authenticate.  The authentication token is written to a file under the Vault agent
install directory (by default, `/opt/vault/data/vault-token`), which only the `vault` user has access
to after installation.

**Note**: To keep this example as simple to deploy and test as possible and because we are
focusing on authentication, it deploys the Vault cluster into your default VPC and default subnets,
 all of which are publicly accessible. This is OK for learning and experimenting, but for
production usage, we strongly recommend deploying the Vault cluster into the private subnets
of a custom VPC.

## Running this example
You will need to create an [Amazon Machine Image (AMI)][ami] that has both Vault and Consul
installed, which you can do using the [vault-consul-ami example][vault_consul_ami]). All the EC2
Instances in this example (including the EC2 Instance that authenticates to Vault) install
either [Dnsmasq][dnsmasq] (via the [install-dnsmasq module][dnsmasq_module])
or [setup-systemd-resolved][setup_systemd_resolved] (in the case of Ubuntu 18.04)
so that all DNS queries for `*.consul` will be directed to the
Consul Server cluster. Because Consul has knowledge of all the Vault nodes (and in
some cases, of other services as well), this setup allows the EC2 Instance to use
Consul's DNS server for service discovery, and thereby to discover the IP addresses
of the Vault nodes.


### Quick start

1. `git clone` this repo to your computer.
1. Build a Vault and Consul AMI. See the [vault-consul-ami example][vault_consul_ami] documentation for
   instructions. Make sure the `install_auth_signing_script` variable is `true`.
   Make sure to note down the ID of the AMI.
1. Install [Terraform](https://www.terraform.io/).
1. Open `variables.tf`, set the environment variables specified at the top of the file, and fill in any other variables
   that don't have a default. Put the AMI ID you previously took note into the `ami_id` variable.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script][examples_helper] to
   print out the IP addresses of the Vault server and some example commands you can run to interact with the cluster:
   `../vault-examples-helper/vault-examples-helper.sh`.
1. Run `curl <auth-instance-ip>:8080` to check if the client instance is fetching the secret from Vault properly


[auto_auth]: https://www.vaultproject.io/docs/agent/autoauth/index.html
[dnsmasq_module]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[setup_systemd_resolved]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/setup-systemd-resolved
[examples_helper]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh
[iam_auth]: https://www.vaultproject.io/docs/auth/aws.html#iam-auth-method
[iam_example]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-iam-auth
[vault_cluster]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster
[vault_consul_ami]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami
