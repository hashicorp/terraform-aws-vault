# Vault Examples Helper

This folder contains a helper script called `vault-examples-helper.sh` for working with the 
[vault-cluster-private](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) and [the root example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/root-example) 
examples. After running `terraform apply` on one of the examples, if you run  `vault-examples-helper.sh`, it will 
automatically:

1. Wait for the Vault server cluster to come up.
1. Print out the IP addresses of the Vault servers.
1. Print out some example commands you can run against your Vault servers.

Please note that this helper script only works because the examples deploy into your default VPC and default subnets.
As a result, Vault is publicly accessible. This is OK for testing and learning, but for production usage, we strongly 
recommend running Vault in private subnets of a custom VPC.
