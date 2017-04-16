# Vault Examples Helper

This folder contains a helper script called `vault-examples-helper.sh` for working with the 
[vault-cluster-private](/examples/vault-cluster-private) and [vault-clsuter-public](/examples/vault-cluster-public) 
examples. After running `terraform apply` on one of the examples, if you run  `vault-examples-helper.sh`, it will 
automatically:

1. Wait for the Vault server cluster to come up.
1. Print out the IP addresses of the Vault servers.
1. Print out some example commands you can run against your Vault servers.


