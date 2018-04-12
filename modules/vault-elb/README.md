# Vault Elastic Load Balancer

This folder contains a [Terraform](https://www.terraform.io/) module that can be used to deploy an [Elastic Load
Balancer (ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) in front of the Vault cluster
from the [vault-cluster module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster). This is useful if you need to access Vault from the public
Internet. Note that for most users, we recommend NOT making Vault accessible from the public Internet and using
DNS to access your Vault cluster instead (see the [install-dnsmasq
module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq) for details).




## How do you use this module?

This folder defines a [Terraform module](https://www.terraform.io/docs/modules/usage.html), which you can use in your
code by adding a `module` configuration and setting its `source` parameter to URL of this folder:

```hcl
module "vault_elb" {
  # Use version v0.0.1 of the vault-elb module
  source = "github.com/hashicorp/terraform-aws-vault//modules/vault-elb?ref=v0.0.1"

  vault_asg_name = "${module.vault_cluster.asg_name}"

  # ... See vars.tf for the other parameters you must define for the vault-cluster module
}

# Configure the Vault cluster to use the ELB
module "vault_cluster" {
  # Use version v0.0.1 of the vault-elb module
  source = "github.com/hashicorp/terraform-aws-vault//modules/vault-cluster?ref=v0.0.1"

  # ... (other params omitted) ...
}
```

Note the following parameters:

* `source`: Use this parameter to specify the URL of the vault-elb module. The double slash (`//`) is intentional
  and required. Terraform uses it to specify subfolders within a Git repo (see [module
  sources](https://www.terraform.io/docs/modules/sources.html)). The `ref` parameter specifies a specific Git tag in
  this repo. That way, instead of using the latest version of this module from the `master` branch, which
  will change every time you run Terraform, you're using a fixed version of the repo.

* `vault_asg_name`: Setting this parameter to the name of the Autoscaling group created by the
  [vault-cluster module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster)
  tells it to register each server with the ELB when it is booting.

You can find the other parameters in [vars.tf](vars.tf).

Check out the [vault-cluster-public example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-public) for working sample code.




## How is the ELB configured?

The ELB in this module is configured as follows:

1. **TCP Passthrough**: The ELB does NOT attempt to terminate SSL, as your Vault servers should do that themselves.
   This ensures that all Vault information is encrypted end-to-end, with no middle man (including AWS) able to read
   the contents. It also allows your Vault servers to do [mutual TLS
   authentication](https://en.wikipedia.org/wiki/Mutual_authentication) so that Vault clients verify the server's
   certificate and the Vault server verifies the client's certificate.

1. **Listeners**: The ELB only listens on one port (default: 443) and forwards the requests to Vault's API port
   (default: 8200).

1. **Health Check**: The ELB uses the [/sys/health endpoint](https://www.vaultproject.io/api/system/health.html) on
   your Vault servers, with the `standbyok` flag set to `true`, as a health check endpoint. This way, the ELB will see
   any primary or standby Vault node that is unsealed as healthy and route traffic to it.

1. **DNS**: If you set the `create_dns_entry` variable to `true`, this module will create a DNS A Record in [Route
   53](https://aws.amazon.com/route53/) that points your specified `domain_name` at the ELB. This allows you to use
   this domain name to access the ELB. Note that the TLS certificate you use with Vault should be configured with this
   same domain name!
