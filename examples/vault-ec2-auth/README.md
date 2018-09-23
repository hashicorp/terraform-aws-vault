# Vault authentication using EC2 metadata example

This example shows how to use the metadata from an EC2 instance to authenticate
to a [vault cluster][vault_cluster].

Vault provides multiple [auth methods][auth_methods] such as Username & Password, GitHub
etc. Among those methods you will find AWS. The way it works is that it
understands [AWS][aws_auth] as a trusted third party, and relies on AWS itself for affirming
if an authentication source such as an EC2 Instance or other resources like a
Lambda Function are legitimate sources or not. Basically, if AWS trusts the
origin, then so do we.

There are currently two ways an AWS resource can authenticatate: `ec2` and `iam`. In
this example, we will explore the first option.


## EC2 Auth

EC2 auth is a process in which Vault relies on information about an EC2 instance
trying to assume a desired authentication role. For different resources that are
not EC2 instances, please refer to the `iam` auth method.

The workflow is that the client trying to authenticate itself will send a
signature in its login request, Vault verifies the signature with AWS, checks
against a predefined authentication role, then returns a client token that the
client can use for making future requests to vault. More details about the
signature and how this works at the section [authenticating from an
instance](#authenticating-from-an-instance)

It is important to notice that, once the server receives a login request with a
signature, to be able to verify it against AWS and check the instance
metadata information, the vault server needs to be allowed to do certain
operations on AWS such as `ec2:DescribeInstances`. On this example, we use the
same [policy][consul_policy] defined for `Consul` since it also has these
permissions.


### Configuring a Vault server

Before we try to authenticate, we must be sure that the server is prepared to
receive requests. Besides enabling the AWS auth method with `vault auth enable
aws` (after making sure that server is already initialized and unsealed), it is
necessary to define the correct policies and roles for authenticating.

Policies are rules that grant or forbid access and actions to certain paths in
Vault. You can read more about them [here][policies_doc]. With one or more
policies on hand, you can then finally create the authentication role.

When creating a role, you can define which set of policies are atteched to that
role, how you wish to configure reauthentication and expiration of tokens issues
by this role, as well as define which set of criteria related to the EC2 instance
metadata upon which you wish to allow access.

In our example we create a simple policy that allows writing and reading from a
namespaced backend and then create a role that allows authentication from all
instances with a specific `ami id`. You can read more about role
creation and check which other instance metadata you can use on auth [here][create_role].


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

See the whole example script at [user-data-vault.sh](user-data-vault.sh).

### Authenticating from an instance

The signature used to authenticate to Vault is a PKCS7 certificate that is part of the AWS
[Instance Identity document][instance_identity]. This certificate can be fetched from the EC2
metadata API with `$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7 | tr -d '\n')`
and will then be part of the body of data sent with the login request.

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
reason, when the instance does its first login, it also receives a crytographic
`nonce` and this `nonce` has to be provided in the future login attempts.


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

To see the full script for authenticating check the [client user data
script](user-data-auth-client.sh).


[vault_cluster]: ../../modules/vault-cluster
[policies_doc]: https://www.vaultproject.io/docs/concepts/policies.html
[auth_methods]: https://www.vaultproject.io/docs/auth/index.html
[create_role]: https://www.vaultproject.io/api/auth/aws/index.html#create-role
[consul_policy]: https://github.com/hashicorp/terraform-aws-consul/blob/master/modules/consul-iam-policies/main.tf
[instance_identity]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
[aws_auth]:https://www.vaultproject.io/docs/auth/aws.html
[nonce]: https://www.vaultproject.io/docs/auth/aws.html#client-nonce
