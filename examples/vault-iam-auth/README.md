# Vault authentication using AWS IAM role example

This example shows how to use the AWS IAM role attached to a resource to authenticate
to a [vault cluster][vault_cluster].

Vault provides multiple ways to authenticate a human or machine to Vault, known as
[auth methods][auth_methods]. For example, a human can authenticate with a Username
& Password or with GitHub.

Among those methods you will find [AWS][aws_auth]. The way it works is that Vault
understands AWS as a trusted third party, and relies on AWS itself for affirming
if an authentication source such as an EC2 Instance or other resources like a Lambda
Function are legitimate sources or not.

There are currently two ways an AWS resource can authenticatate to Vault: `ec2` and `iam`.
In this example, we demonstrate the [AWS IAM Auth Method][iam_auth].

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


### Vault Authentication using IAM user or role

IAM auth is a process in which Vault leverages AWS STS (Security Token Service) to
identify the AWS IAM principal (user or role) attached to an AWS resource such as
an ECS Task or a Lambda Function that originates the login request. You can still
use the `iam` method for EC2 instances attached to a role, like we do in this example,
but for a login method specifically for EC2 instances, please refer to the
[`ec2` auth method example][ec2_example].

The workflow is that the client trying to authenticate will create a request to
the method `GetCallerIdentity` of the AWS STS API (but not yet send it). This
method basically answers the question "Who am I?". This request is then signed
with the AWS credentials of the client. The signed result is then sent with the
login request to the Vault Server. When the Vault server receives a login request
with the `iam` method, it can execute the STS request without actually knowing
the contents of the signed part. It then receives a response from STS identifying
who signed it, which the Vault Server then can check against the ARN of the IAM
principal bounded to a previously created Vault Role and decide if it should be
allowed to authenticate or not.

![auth diagram][auth_diagram]

It is important to notice that, when the Vault Server receives this encrypted STS
API request attached to a login request, to be able to execute it and perform the
login, the cluster needs to have AWS Policies that will allow the cluster to execute
the following actions: `iam:GetRole` or `iam:GetUser`, and `sts:GetCallerIdentity`.


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
Role and for how long tokens issued for that Role will be valid.

In our example we create a simple Vault Policy that allows writing and reading from
secrets in the path `secret` namespaced with the prefix `example_`, and then create
a Vault Role that allows authentication from AWS resources attached to a certain IAM Role.
You can read more about Role creation and check which other configurations you can
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


### Authenticating from a client

#### With an HTTP request

The [vault-consul-ami][vault_consul_ami] includes a [python script][py_sign] called
`sign-request.py`. We use python here instead of bash to take advantage of the
`boto3` AWS SDK library. This script is a modified version of the Python 2.x example
posted by J. Thompson, the author of Vault's IAM auth method, at the Vault mailing
list. It uses `boto3` to create a request to the AWS Security Token Service API
with the action "GetCallerIdentity" and then signs the request using the AWS credentials.
The same pattern should work with the AWS SDK in any other supported language such
as Go, Java or Ruby, for example. For more details on the IAM auth method, there's
a talk by J. Thompson called [Deep Dive into Vault's AWS Auth Backend][talk].

```bash
signed_request=$(python /opt/vault/scripts/sign-request.py vault.service.consul)
```

Once we have the encrypted request created by the python script, we can pass it
in the body of the login request we will send to the Vault Server.

```
iam_request_url=$(echo $signed_request | jq -r .iam_request_url)
iam_request_body=$(echo $signed_request | jq -r .iam_request_body)
iam_request_headers=$(echo $signed_request | jq -r .iam_request_headers)


data=$(cat <<EOF
{
  "role":"$VAULT_ROLE_NAME",
  "iam_http_request_method": "POST",
  "iam_request_url": "$iam_request_url",
  "iam_request_body": "$iam_request_body",
  "iam_request_headers": "$iam_request_headers"
}
EOF
)

curl --request POST --data "$data" https://vault.service.consul:8200/v1/auth/aws/login"
```

After sending the login request to Vault, Vault will execute the STS request to
verify the client's identity with AWS and return a JSON object with your login
information containing the `client_token`. The client token is an ephemeral token
that you will send with your future operations requests to Vault. It can expire,
be rotated, or become invalid for some other reason and you might be required to
authenticate again.

To see the full example script for authenticating, check the [client user data script][user_data_auth_client].


#### With Vault cli tool

If vault cli is installed we can perform the login operation with it. The `VAULT_ADDR`
environment variable has to be set and you need to have AWS credentials in some form.
The vault cli will look for credentials configured in the standard locations such as
environment variables, ~/.aws/credentials, IAM instance profile, or ECS task role, in
that order. The way the it works is the same as with an HTTP request. The vault cli
tool uses the golang AWS SDK to the create the STS API request and sign it with the
credentials for you. It's important to note that the `role` value being passed is
the Vault Role name, not the AWS IAM Role name.

```bash
export VAULT_ADDR=https://vault.service.consul:8200
vault login -method=aws header_value=vault.service.consul role=vault-role-name
```

[ami]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html
[auth_diagram]: https://raw.githubusercontent.com/hashicorp/terraform-aws-vault/master/examples/vault-iam-auth/images/iam-auth.png
[auth_methods]: https://www.vaultproject.io/docs/auth/index.html
[aws_auth]:https://www.vaultproject.io/docs/auth/aws.html
[consul_policy]: https://github.com/hashicorp/terraform-aws-consul/blob/master/modules/consul-iam-policies/main.tf
[create_role]: https://www.vaultproject.io/api/auth/aws/index.html#create-role
[dnsmasq_module]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[setup_systemd_resolved]: https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/setup-systemd-resolved
[ec2_example]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-ec2-auth
[examples_helper]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh
[iam_auth]: https://www.vaultproject.io/docs/auth/aws.html#iam-auth-method
[policies_doc]: https://www.vaultproject.io/docs/concepts/policies.html
[py_sign]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami/auth/sign-request.py
[talk]: https://www.hashicorp.com/resources/deep-dive-vault-aws-auth-backend
[user_data_auth_client]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-iam-auth/user-data-auth-client.sh
[user_data_vault]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-iam-auth/user-data-vault.sh
[vault_cluster]: https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster
[vault_consul_ami]: https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami
