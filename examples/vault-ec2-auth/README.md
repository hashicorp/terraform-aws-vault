# Vault authentication using EC2 metadata example

This example shows how to use the metadata from an EC2 instance to authenticate
to a [vault cluster][vault_cluster].

Vault provides multiple ways to authenticate a human or machine to Vault, known as
[auth methods][auth_methods]. For example, a human can authenticate with a Username
& Password or with GitHub.

Among those methods you will find [AWS][aws_auth]. The way it works is that Vault
understands AWS as a trusted third party, and relies on AWS itself for affirming
if an authentication source such as an EC2 Instance or other resources like a Lambda
Function are legitimate sources or not.

There are currently two ways an AWS resource can authenticatate to Vault: `ec2` and `iam`.
In this example, we demonstrate the [AWS EC2 Auth Method][ec2_auth].

**Note**: To keep this example as simple to deploy and test as possible and because we are
focusing on authentication, it deploys the Vault cluster into your default VPC and default subnets,
 all of which are publicly accessible. This is OK for learning and experimenting, but for
production usage, we strongly recommend deploying the Vault cluster into the private subnets
of a custom VPC.

## Running this example
You will need to create an [Amazon Machine Image (AMI)][ami] that has both Vault and Consul
installed, which you can do using the [vault-consul-ami example][vault_consul_ami]). All the EC2
Instances in this example (including the EC2 Instance that authenticates to Vault) install
[Dnsmasq][dnsmasq] (via the [install-dnsmasq module][dnsmasq_module]) or
[setup-systemd-resolved][setup_systemd_resolved] (in the case of Ubuntu 18.04) so that all DNS queries
for `*.consul` will be directed to the Consul Server cluster. Because Consul has knowledge of
all the Vault nodes (and in some cases, of other services as well), this setup allows the EC2
Instance to use Consul's DNS server for service discovery, and thereby to discover the IP addresses
of the Vault nodes.

### Quick start

1. `git clone` this repo to your computer.
1. Build a Vault and Consul AMI. See the [vault-consul-ami example][vault_consul_ami] documentation for
   instructions. Make sure to note down the ID of the AMI.
1. Install [Terraform](https://www.terraform.io/).
1. Open `variables.tf`, set the environment variables specified at the top of the file, and fill in any other variables that
   don't have a default. Put the AMI ID you previously took note into the `ami_id` variable.
1. Run `terraform init`.
1. Run `terraform apply`.
1. Run the [vault-examples-helper.sh script][examples_helper] to
   print out the IP addresses of the Vault server and some example commands you can run to interact with the cluster:
   `../vault-examples-helper/vault-examples-helper.sh`.
1. Run `curl <auth-instance-ip>:8080` to check if the client instance is fetching the secret from Vault properly


## EC2 Auth

EC2 auth is a process in which Vault relies on information about an EC2 instance
trying to assume a desired authentication role. For different resources that are
not EC2 instances, please refer to the [`iam` auth method example][iam_example].

The workflow is that the client trying to authenticate itself will send a
signature in its login request, Vault verifies the signature with AWS, checks
against a predefined authentication role, then returns a client token that the
client can use for making future requests to vault. More details about the
signature and how this works at the section [authenticating from an
instance](#authenticating-from-an-instance)

![auth diagram][auth_diagram]

It is important to notice that, once the server receives a login request with a
signature, to be able to verify it against AWS and check the instance
metadata information, the vault server needs to be allowed to do certain
operations on AWS such as `ec2:DescribeInstances`. On this example, we use the
same [policy][consul_policy] defined for `Consul` since it also has these
permissions.


### Configuring a Vault server

Before we try to authenticate, we must be sure that the Vault Server is configured
properly and prepared to receive requests. First, we must make sure the Vault server
has been initialized (using `vault operator init`) and unsealed (using `vault operator unseal`).
Next, we must enable Vault to support the AWS auth method (using `vault auth enable aws`).
After that, we enable the Vault kv secrets engine at the path `secret` (note that this engine
was enabled by default in previous versions < 1.1.0).  Finally, we must define the correct
Vault Policies and Roles to declare which IAM Principals will have access to which resources
in Vault.

[Policies][policies_doc] are rules that grant or forbid access and actions to certain paths in
Vault. With one or more policies on hand, you can then finally create the authentication role.

When you create a Role in Vault, you define the Policies that are attached to that
Role, how principals who assume that Role will re-authenticate, and for how long
tokens issued for that role will be valid. When your Role uses the EC2 AWS Auth
method, you also specify which of the EC2 Instance Metadata properties will be
required by the principal (in this case, the EC2 Instance) in order to successfully
authenticate.

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
  auth_type=ec2 \
  policies=example-policy \
  max_ttl=500h \
  bound_ami_id=$ami_id
```

See the whole example script at [user-data-vault.sh][user_data_vault].


### Authenticating from an instance

The signature used to authenticate to Vault is a PKCS7 certificate that is part of the AWS
[Instance Identity Document][instance_identity]. This certificate can be fetched from the EC2
metadata API with `$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7 | tr -d '\n')`
and will then be part of the body of data sent with the login request.

The Instance Identity Document describes various features of the EC2 Instance like its Instance Type,
region, IAM Role and a "signature" for this document that is signed by AWS. The signature can be used
to prove that the Instance Identity Document was produced by AWS, and not a malicious third party. By
sending the Instance Identity Document and signature to Vault, you are proving to Vault that you are
an EC2 Instance that genuinely has the properties described in the Instance Identity Document. Vault
can then use these properties to help decide whether to authenticate you.

```bash
data=$(cat <<EOF
{
  "role": "example-role",
  "pkcs7": "$pkcs7"
}
EOF
)
curl --request POST --data "$data" "https://vault.service.consul:8200/v1/auth/aws/login"
```

After sending the login request to Vault, Vault will verify it against AWS and
return a JSON object with your login information. This JSON contains two
important values: the `client_token` and the `nonce`. The client token is an
ephemeral token that you will send with your future operations requests to
Vault. It can expire, be rotated, or become invalid for some other reason and
you will be required to authenticate again.

However, as a security measure, Vault operates with a TOFU (Trust on First Use)
mechanism. That means that once an instance logins, all other logins with the
same signature will fail. The reason is to prevent unintended logins in case the
PKCS7 signature gets compromised by another process in the instance. For this
reason, when the instance does its first login, it also receives a cryptographic
`nonce` (number used once) and this `nonce` has to be provided in the future
login attempts. Only one unique principal (i.e. one unique EC2 Instance ID) can
use that nonce.

```bash
data=$(cat <<EOF
{
  "role": "example-role",
  "pkcs7": "$pkcs7",
  "nonce": "$nonce"
}
EOF
)
curl --request POST --data "$data" "https://vault.service.consul:8200/v1/auth/aws/login"
```

It is up to the client to decide how it handles the nonce. To read more about
it, refer to the [Vault documentation on client nonce][nonce].

To see the full script for authenticating check the [client user data script][user_data_auth_client].


[ami]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html
[auth_methods]: https://www.vaultproject.io/docs/auth/index.html
[auth_diagram]: https://raw.githubusercontent.com/hashicorp/terraform-aws-vault/master/examples/vault-ec2-auth/images/ec2-auth.png
[aws_auth]:https://www.vaultproject.io/docs/auth/aws.html
[consul_policy]: https://github.com/hashicorp/terraform-aws-consul/blob/master/modules/consul-iam-policies/main.tf
[create_role]: https://www.vaultproject.io/api/auth/aws/index.html#create-role
[dnsmasq_module]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[setup_systemd_resolved]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/setup-systemd-resolved
[ec2_auth]: https://www.vaultproject.io/docs/auth/aws.html#ec2-auth-method
[examples_helper]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh
[iam_example]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-iam-auth
[instance_identity]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
[nonce]: https://www.vaultproject.io/docs/auth/aws.html#client-nonce
[policies_doc]: https://www.vaultproject.io/docs/concepts/policies.html
[user_data_auth_client]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-ec2-auth/user-data-auth-client.sh
[user_data_vault]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-ec2-auth/user-data-vault.sh
[vault_cluster]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster
[vault_consul_ami]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami
