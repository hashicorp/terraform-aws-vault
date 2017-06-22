# Vault AWS Blueprint

This repo contains a Blueprint for how to deploy a [Vault](https://www.vaultproject.io/) cluster on 
[AWS](https://aws.amazon.com/) using [Terraform](https://www.terraform.io/). Vault is an open source tool for managing
secrets. This Blueprint uses [S3](https://aws.amazon.com/s3/) as a [storage 
backend](https://www.vaultproject.io/docs/configuration/storage/index.html) and a [Consul](https://www.consul.io) 
server cluster as a [high availability backend](https://www.vaultproject.io/docs/concepts/ha.html):

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
    
* [vault-elb](/modules/vault-elb): Configures an [Elastic Load Balancer 
  (ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) in front of Vault if you need to access it
  from the public Internet.
   
* [private-tls-cert](/modules/private-tls-cert): Generate a private TLS certificate for use with a private Vault 
  cluster.
   
* [update-certificate-store](/modules/update-certificate-store): Add a trusted, CA public key to an OS's 
  certificate store. This allows you to establish TLS connections to services that use this TLS certs signed by this
  CA without getting x509 certificate errors.
   



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

This Blueprint is maintained by [Gruntwork](http://www.gruntwork.io/). If you're looking for help or commercial 
support, send an email to [blueprints@gruntwork.io](mailto:blueprints@gruntwork.io?Subject=Vault%20Blueprint). 
Gruntwork can help with:

* Setup, customization, and support for this Blueprint.
* Blueprints for other types of infrastructure, such as VPCs, Docker clusters, databases, and continuous integration.
* Blueprints that meet compliance requirements, such as HIPAA.
* Consulting & Training on AWS, Terraform, and DevOps.



## How do you use this Blueprint?

Each Blueprint has the following folder structure:

* [modules](/modules): This folder contains the reusable code for this Blueprint, broken down into one or more modules.
* [examples](/examples): This folder contains examples of how to use the modules.
* [test](/test): Automated tests for the modules and examples.

Click on each of the modules above for more details.

To deploy Vault with this Blueprint, you will need to deploy two separate clusters: one to run 
[Consul](https://www.consul.io/) servers (which Vault uses as a [high availability 
backend](https://www.vaultproject.io/docs/concepts/ha.html)) and one to run Vault servers. 

To deploy the Consul server cluster, use the [Consul AWS Blueprint](https://github.com/gruntwork-io/consul-aws-blueprint). 

To deploy the Vault cluster:

1. Create an AMI that has Vault installed (using the [install-vault module](/modules/install-vault)) and the Consul
   agent installed (using the [install-consul 
   module](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/install-consul)). Here is an 
   [example Packer template](/examples/vault-consul-ami). 
   
   If you are just experimenting with this Blueprint, you may find it more convenient to use one of our official public AMIs:
   - [Latest Ubuntu 16 AMIs](/_docs/ubuntu16-ami-list.md).
   - [Latest Amazon Linux AMIs](/_docs/amazon-linux-ami-list.md).
   
   **WARNING! Do NOT use these AMIs in your production setup. In production, you should build your own AMIs in your 
     own AWS account.**

1. Deploy that AMI across an Auto Scaling Group in a private subnet using the Terraform [vault-cluster 
   module](/modules/vault-cluster). 

1. Execute the [run-consul script](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/run-consul)
   with the `--client` flag during boot on each Instance to have the Consul agent connect to the Consul server cluster. 

1. Execute the [run-vault](/modules/run-vault) script during boot on each Instance to create the Vault cluster. 

1. If you only need to access Vault from inside your AWS account (recommended), run the [install-dnsmasq 
   module](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/install-dnsmasq) on each server, and 
   that server will be able to reach Vault using the Consul Server cluster as the DNS resolver (e.g. using an address 
   like `vault.service.consul`). See the [vault-cluster-private example](/examples/vault-cluster-private) for working 
   sample code.

1. If you need to access Vault from the public Internet, deploy the [vault-elb module](/modules/vault-elb) in a public 
   subnet and have all requests to Vault go through the ELB. See the [vault-cluster-public 
   example](/examples/vault-cluster-public) for working sample code.

1. Head over to the [How do you use the Vault cluster?](/modules/vault-cluster#how-do-you-use-the-vault-cluster) guide
   to learn how to initialize, unseal, and use Vault.

 
 



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

