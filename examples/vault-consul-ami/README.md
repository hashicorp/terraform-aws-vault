# Vault and Consul AMI

This folder shows an example of how to use the [install-vault module](/modules/install-vault) from this Blueprint and 
the [install-consul](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/install-consul)
and [install-dnsmasq](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/install-dnsmasq) modules
from the Consul AWS Blueprint with [Packer](https://www.packer.io/) to create [Amazon Machine Images 
(AMIs)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) that have Vault and Consul installed on top of:
 
1. Ubuntu 16.04
1. Amazon Linux

You can use this AMI to deploy a [Vault cluster](https://www.vaultproject.io/) by using the [vault-cluster
module](/modules/vault-cluster). This Vault cluster will use Consul as its storage backend, so you can also use the 
same AMI to deploy a separate [Consul server cluster](https://www.consul.io/) by using the [consul-cluster 
module](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/consul-cluster). 

Check out the [vault-cluster-private](/examples/vault-cluster-private) and 
[vault-cluster-public](/examples/vault-cluster-public) examples for working sample code. For more info on Vault 
installation and configuration, check out the [install-vault](/modules/install-vault) documentation.



## Quick start

To build the Vault and Consul AMI:

1. `git clone` this repo to your computer.

1. Install [Packer](https://www.packer.io/).

1. Configure your AWS credentials using one of the [options supported by the AWS 
   SDK](http://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html). Usually, the easiest option is to
   set the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.

1. Use the [private-tls-cert module](/modules/private-tls-cert) to generate a CA cert and public and private keys for a 
   TLS cert: 
   
    1. Set the `dns_names` parameter to `vault.service.consul`. If you're using the [vault-cluster-public
       example](/examples/vault-cluster-public) and want a public domain name (e.g. `vault.example.com`), add that 
       domain name here too.
    1. Set the `ip_addresses` to `127.0.0.1`. 
    1. For production usage, you should take care to protect the private key by encrypting it (see [Using TLS 
       certs](/modules/private-tls-cert#using-tls-certs) for more info). 

1. Update the `variables` section of the `vault-consul.json` Packer template to specify the AWS region, Vault 
   version, Consul version, and the paths to the TLS cert files you just generated. 

1. Run `packer build vault-consul.json`.

When the build finishes, it will output the IDs of the new AMIs. To see how to deploy one of these AMIs, check out the 
[vault-cluster-private](/examples/vault-cluster-private) and [vault-cluster-public](/examples/vault-cluster-public) 
examples.




## Creating your own Packer template for production usage

When creating your own Packer template for production usage, you can copy the example in this folder more or less 
exactly, except for one change: we recommend replacing the `file` provisioner with a call to `git clone` in the `shell` 
provisioner. Instead of:

```json
{
  "provisioners": [{
    "type": "file",
    "source": "{{template_dir}}/../../../vault-aws-blueprint",
    "destination": "/tmp"
  },{
    "type": "shell",
    "inline": [
      "/tmp/vault-aws-blueprint/modules/install-vault/install-vault --version {{user `vault_version`}}"
    ],
    "pause_before": "30s"
  }]
}
```

Your code should look more like this:

```json
{
  "provisioners": [{
    "type": "shell",
    "inline": [
      "git clone --branch <BLUEPRINT_VERSION> https://github.com/gruntwork-io/vault-aws-blueprint.git /tmp/vault-aws-blueprint",
      "/tmp/vault-aws-blueprint/modules/install-vault/install-vault --version {{user `vault_version`}}"
    ],
    "pause_before": "30s"
  }]
}
```

You should replace `<BLUEPRINT_VERSION>` in the code above with the version of this blueprint that you want to use (see
the [Releases Page](../../releases) for all available versions). That's because for production usage, you should always
use a fixed, known version of this Blueprint, downloaded from the official Git repo. On the other hand, when you're 
just experimenting with the Blueprint, it's OK to use a local checkout of the Blueprint, uploaded from your own 
computer.