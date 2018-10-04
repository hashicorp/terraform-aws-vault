# Vault authentication using AWS IAM role example

This example shows how to use the AWS IAM role attached to a resource to authenticate
to a [vault cluster][vault_cluster].

Vault provides multiple [auth methods][auth_methods] such as Username & Password, GitHub
etc. Among those methods you will find AWS. The way it works is that it
understands [AWS][aws_auth] as a trusted third party, and relies on AWS itself for affirming
if an authentication source such as an EC2 Instance or other resources like a
Lambda Function are legitimate sources or not. Basically, if AWS trusts the
origin, then so do we.

Vault provides multiple ways to authenticate a human or machine to Vault, known as
[auth methods][auth_methods]. For example, a human can authenticate with a Username
& Password or with GitHub. In this example, we demonstrate the [AWS Auth Method][aws_auth].

The way it works is that Vault understands [AWS][aws_auth] as a trusted third party, and
relies on AWS itself for affirming if an authentication source such as an EC2 Instance or
other resources like a Lambda Function are legitimate sources or not. Basically, if AWS
trusts the origin, then so do we.

There are currently two ways an AWS resource can authenticatate: `ec2` and `iam`. In
this example, we will explore the second option.

**Note**: To keep this example as simple to deploy and test as possible and because we are
focusing on authentication, it deploys the Vault cluster into your default VPC and default subnets,
 all of which are publicly accessible. This is OK for learning and experimenting, but for
production usage, we strongly recommend deploying the Vault cluster into the private subnets
of a custom VPC.

## Running this example
You will need to create an [Amazon Machine Image (AMI)][ami] that has both Vault and Consul
installed, which you can do using the [vault-consul-ami example][vault_consul_ami]). All the EC2
Instances in this example (including the EC2 Instance that authenticates to Vault) install
[Dnsmasq][dnsmasq] (via the [install-dnsmasq module][dnsmasq_module]) so that all DNS queries
for `*.consul` will be directed to the Consul Server cluster. Because Consul has knowledge of
all the Vault nodes (and in some cases, of other services as well), this setup allows the EC2
Instance to use Consul's DNS server for service discovery, and thereby to discover the IP addresses
of the Vault nodes.


### Quick start

1. `git clone` this repo to your computer.
1. Build a Vault and Consul AMI. See the [vault-consul-ami example][vault_consul_ami] documentation for
   instructions. Make sure to note down the ID of the AMI.
1. Install [Terraform](https://www.terraform.io/).
1. Open `vars.tf`, set the environment variables specified at the top of the file, and fill in any other variables that
   don't have a default. Put the AMI ID you previously took note into the `ami_id` variable.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script][examples_helper] to
   print out the IP addresses of the Vault server and some example commands you can run to interact with the cluster:
   `../vault-examples-helper/vault-examples-helper.sh`.
1. Run `curl <auth-instance-ip>:8080` to check if the client instance is fetching the secret from Vault properly


### Vault Authentication using IAM user or role

To read more about Vault IAM auth, refer to [Vault AWS Auth documentation][aws_auth].

### Configuring a Vault server

Before we try to authenticate, we must be sure that the Vault Server is configured
properly and prepared to receive requests. First, we must make sure the Vault server
has been initialized (using `vault operator init`) and unsealed (using `vault operator unseal`).
Next, we must enable Vault to support the AWS auth method (using `vault auth enable aws`).
Finally, we must define the correct Vault Policies and Roles to declare which EC2
Instances will have access to which resources in Vault.

[Policies][policies_doc] are rules that grant or forbid access and actions to certain paths in
Vault. With one or more policies on hand, you can then finally create the authentication role.

When you create a Role in Vault, you define the Policies that are attached to that
Role, how principals who assume that Role will re-authenticate, and for how long
tokens issued for that role will be valid.

In our example we create a simple Vault Policy that allows writing and reading from
secrets in the path `secret` namespaced with the prefix `example_`, and then create
a Vault Role that allows authentication from all instances with a specific `ami id`.
You can read more about Role creation and check which other instance metadata you can
use on auth [here][create_role].


```bash
vault policy write "example-policy" -<<EOF
path "secret/example_*" {
  capabilities = ["create", "read"]
}
EOF

vault write \
  auth/aws/role/example-role
  auth_type=iam \
  policies=example-policy \
  max_ttl=500h \
  bound_iam_principal_arn=<ARN>
```

See the whole example script at [user-data-vault.sh][user_data_vault].


### Authenticating from an instance

To see the full script for authenticating check the [client user data script][user_data_auth_client].


[ami]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html
[auth_methods]: https://www.vaultproject.io/docs/auth/index.html
[aws_auth]:https://www.vaultproject.io/docs/auth/aws.html
[consul_policy]: https://github.com/hashicorp/terraform-aws-consul/blob/master/modules/consul-iam-policies/main.tf
[create_role]: https://www.vaultproject.io/api/auth/aws/index.html#create-role
[dnsmasq_module]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[examples_helper]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh
[policies_doc]: https://www.vaultproject.io/docs/concepts/policies.html
[user_data_auth_client]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-iam-auth/user-data-auth-client.sh
[user_data_vault]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-iam-auth/user-data-vault.sh
[vault_cluster]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster
[vault_consul_ami]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami
