# Vault AWS Blueprint

This repo contains a Blueprint for how to deploy a [Vault](https://www.vaultproject.io/) cluster on 
[AWS](https://aws.amazon.com/) using [Terraform](https://www.terraform.io/). Vault is an open source tool for managing
secrets. This Blueprint uses [Consul](https://www.consul.io) as a [storage 
backend](https://www.vaultproject.io/docs/configuration/storage/index.html):

![Vault architecture](/_docs/architecture.png)

This Blueprint includes:

* [install-vault](/modules/install-valut): This module can be used to install Vault. It can be used in a 
  [Packer](https://www.packer.io/) template to create a Vault 
  [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html).

* [run-vault](/modules/run-vault): This module can be used to configure and run Vault. It can be used in a 
  [User Data](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-shell-scripts) 
  script to fire up Vault while the server is booting.

* [vault-cluster](/modules/vault-cluster): Terraform code to deploy a cluster of Vault servers using an [Auto Scaling 
  Group](https://aws.amazon.com/autoscaling/).
    
* [vault-dnsmasq](/modules/vault-dnsmasq): Installs Dnsmasq on your servers so you can access Vault using DNS (e.g. 
  using an address like `vault.service.consul`).
    
* [vault-elb](/modules/vault-elb): Configures an [Elastic Load Balancer 
  (ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) in front of Vault if you need to access it
  from the public Internet.
   



## What's a Blueprint?

A Blueprint is a canonical, reusable, best-practices definition for how to run a single piece of infrastructure, such 
as a database or server cluster. Each Blueprint is created primarily using [Terraform](https://www.terraform.io/), 
includes automated tests, examples, and documentation, and is maintained both by the open source community and 
companies that provide commercial support. 

Instead of having to figure out the details of how to run a piece of infrastructure from scratch, you can reuse 
existing code that has been proven in production. And instead of maintaining all that infrastructure code yourself, 
you can leverage the work of the Blueprint community and maintainers, and pick up infrastructure improvements through
a version number bump.
 
 
 
## Who maintains this Blueprint?

This Blueprint is maintained by [Gruntwork](http://www.gruntwork.io/). If you need help or support, send an email to 
[blueprints@gruntwork.io](mailto:blueprints@gruntwork.io?Subject=Vault%20Blueprint). Gruntwork can help with:

* Blueprints for other types of infrastructure, such as VPCs, Docker clusters, databases, and continuous integration.
* Blueprints that meet compliance requirements, such as HIPAA.
* Consulting & Training on AWS, Terraform, and DevOps.



## How do you use this Blueprint?

Each Blueprint has the following folder structure:

* [modules](/modules): This folder contains the reusable code for this Blueprint, broken down into one or more modules.
* [examples](/examples): This folder contains examples of how to use the modules.
* [test](/test): Automated tests for the modules and examples.

Click on each of the modules above for more details.

To deploy a Vault cluster with this Blueprint:

1. Deploy a [Consul](https://www.consul.io/) cluster using the [Consul AWS 
   Blueprint](https://github.com/gruntwork-io/consul-aws-blueprint). This Vault Blueprint uses Consul as its
   [storage backend](https://www.vaultproject.io/docs/configuration/storage/index.html).

1. Create an AMI that has Vault installed (using the [install-vault module](/modules/install-vault)) and the Consul
   agent installed (using the [install-consul 
   module](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/install-consul)). Here is an 
   [example Packer template](/examples/vault-ami).

1. Deploy that AMI across an Auto Scaling Group in a private subnet using the Terraform [vault-cluster 
   module](/modules/vault-cluster). 

1. Execute the [run-consul script](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/run-consul)
   with the `--client` flag during boot on each Instance to have the Consul agent connect to the Consul cluster. 

1. Execute the [run-vault](/modules/run-vault) script during boot on each Instance to create the Vault cluster. 

1. The first time only: Connect to one of the Vault nodes (e.g. via SSH) and run the [vault 
   init](https://www.vaultproject.io/intro/getting-started/deploy.html#initializing-the-vault) command to create the
   unseal keys and root token. For production usage, we **strongly** recommend running the init command with
   [Keybase, PGP, or GPG](https://www.vaultproject.io/docs/concepts/pgp-gpg-keybase.html) to encrypt the unseal keys
   and token. Distribute the unseal keys to your trusted administrators.

1. Every time a Vault node boots: Have each administrator connect to each of the Vault nodes (e.g. via SSH) and run 
   the [unseal command](https://www.vaultproject.io/docs/concepts/seal.html) with their unseal key. Once the proper 
   number of key shards have been entered, your Vault nodes will be unsealed, and your cluster will be ready for use!

If you only need to access Vault from inside your AWS account (recommended), install the [vault-dnsmasq 
module](/modules/vault-dnsmasq) on each server, and that server will be able to reach Vault using DNS (e.g. using an
address like `vault.service.consul`). See the [vault-cluster-private example](/examples/vault-cluster-private) for 
working sample code.

If you need to access Vault from the public Internet, deploy the [vault-elb module](/modules/vault-elb) in a public 
subnet and have all requests to Vault go through the ELB. See the [vault-cluster-public 
example](/examples/vault-cluster-public) for working sample code.


 



## How do I contribute to this Blueprint?

Contributions are very welcome! Check out the [Contribution Guidelines](/CONTRIBUTING.md) for instructions.



## How is this Blueprint versioned?

This Blueprint follows the principles of [Semantic Versioning](http://semver.org/). You can find each new release, 
along with the changelog, in the [Releases Page](../../releases). 

During initial development, the major version will be 0 (e.g., `0.x.y`), which indicates the code does not yet have a 
stable API. Once we hit `1.0.0`, we will make every effort to maintain a backwards compatible API and use the MAJOR, 
MINOR, and PATCH versions on each release to indicate any incompatibilities. 



## License

This code is released under the Apache 2.0 License. Please see [LICENSE](/LICENSE) and [NOTICE](/NOTICE) for more 
details.

