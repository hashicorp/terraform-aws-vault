[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_aws_vault)
# Vault AWS Module

This repo contains a set of modules in the [modules folder](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules) for deploying a [Vault](https://www.vaultproject.io/) cluster on
[AWS](https://aws.amazon.com/) using [Terraform](https://www.terraform.io/). Vault is an open source tool for managing
secrets. By default, this Module uses [Consul](https://www.consul.io) as a [storage
backend](https://www.vaultproject.io/docs/configuration/storage/index.html). You can optionally add an [S3](https://aws.amazon.com/s3/) backend for durability.

![Vault architecture](https://github.com/hashicorp/terraform-aws-vault/blob/master/_docs/architecture.png?raw=true)

This Module includes:

* [install-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault): This module can be used to install Vault. It can be used in a
  [Packer](https://www.packer.io/) template to create a Vault
  [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html).

* [run-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/run-vault): This module can be used to configure and run Vault. It can be used in a
  [User Data](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-shell-scripts)
  script to fire up Vault while the server is booting.

* [vault-cluster](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster): Terraform code to deploy a cluster of Vault servers using an [Auto Scaling
  Group](https://aws.amazon.com/autoscaling/).

* [vault-elb](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-elb): Configures an [Elastic Load Balancer
  (ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) in front of Vault if you need to access it
  from the public Internet.

* [private-tls-cert](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/private-tls-cert): Generate a private TLS certificate for use with a private Vault
  cluster.

* [update-certificate-store](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/update-certificate-store): Add a trusted, CA public key to an OS's
  certificate store. This allows you to establish TLS connections to services that use this TLS certs signed by this
  CA without getting x509 certificate errors.



## How do you use this Module?

This repo has the following structure:

* [modules](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules): This folder contains several standalone, reusable, production-grade modules that you can use to deploy Vault.
* [examples](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples): This folder shows examples of different ways to combine the modules in the `modules` folder to deploy Vault.
* [test](https://github.com/hashicorp/terraform-aws-vault/tree/master/test): Automated tests for the modules and examples.
* [root folder](https://github.com/hashicorp/terraform-aws-vault/tree/master): The root folder is *an example* of how to use the [vault-cluster module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster)
  module to deploy a [Vault](https://www.vaultproject.io/) cluster in [AWS](https://aws.amazon.com/). The Terraform Registry requires the root of every repo to contain Terraform code, so we've put one of the examples there. This example is great for learning and experimenting, but for production use, please use the underlying modules in the [modules folder](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules) directly.

To deploy Vault to production with this repo, you will need to deploy two separate clusters: one to run
[Consul](https://www.consul.io/) servers (which Vault uses as a [storage
backend](https://www.vaultproject.io/docs/configuration/storage/index.html)) and one to run Vault servers.

To deploy the Consul server cluster, use the [Consul AWS Module](https://github.com/hashicorp/terraform-aws-consul).

To deploy the Vault cluster:

1. Create an AMI that has Vault installed (using the [install-vault module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault)) and the Consul
   agent installed (using the [install-consul
   module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-consul)). Here is an
   [example Packer template](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami).

   If you are just experimenting with this Module, you may find it more convenient to use one of our official public AMIs.
   Check out the `aws_ami` data source usage in `main.tf` for how to auto-discover this AMI.

   **WARNING! Do NOT use these AMIs in your production setup. In production, you should build your own AMIs in your
     own AWS account.**

1. Deploy that AMI across an Auto Scaling Group in a private subnet using the Terraform [vault-cluster
   module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster).

1. Execute the [run-consul script](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/run-consul)
   with the `--client` flag during boot on each Instance to have the Consul agent connect to the Consul server cluster.

1. Execute the [run-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/run-vault) script during boot on each Instance to create the Vault cluster.

1. If you only need to access Vault from inside your AWS account (recommended), run the [install-dnsmasq
   module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq) on each server or
   [setup-systemd-resolved](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/setup-systemd-resolved)
   (in the case of Ubuntu 18.04) and 
   that server will be able to reach Vault using the Consul Server cluster as the DNS resolver (e.g. using an address
   like `vault.service.consul`). See the [vault-cluster-private example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) for working
   sample code.

1. If you need to access Vault from the public Internet, deploy the [vault-elb module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-elb) in a public
   subnet and have all requests to Vault go through the ELB. See the [main.tf in the root folder of this repo
   example](https://github.com/hashicorp/terraform-aws-vault/blob/master/main.tf) for working sample code.

1. Head over to the [How do you use the Vault cluster?](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-cluster#how-do-you-use-the-vault-cluster) guide
   to learn how to initialize, unseal, and use Vault.




## What's a Module?

A Module is a canonical, reusable, best-practices definition for how to run a single piece of infrastructure, such
as a database or server cluster. Each Module is created primarily using [Terraform](https://www.terraform.io/),
includes automated tests, examples, and documentation, and is maintained both by the open source community and
companies that provide commercial support.

Instead of having to figure out the details of how to run a piece of infrastructure from scratch, you can reuse
existing code that has been proven in production. And instead of maintaining all that infrastructure code yourself,
you can leverage the work of the Module community and maintainers, and pick up infrastructure improvements through
a version number bump.



## Who maintains this Module?

This Module is maintained by [Gruntwork](http://www.gruntwork.io/). If you're looking for help or commercial
support, send an email to [modules@gruntwork.io](mailto:modules@gruntwork.io?Subject=Vault%20Module).
Gruntwork can help with:

* Setup, customization, and support for this Module.
* Modules for other types of infrastructure, such as VPCs, Docker clusters, databases, and continuous integration.
* Modules that meet compliance requirements, such as HIPAA.
* Consulting & Training on AWS, Terraform, and DevOps.




## How do I contribute to this Module?

Contributions are very welcome! Check out the [Contribution Guidelines](https://github.com/hashicorp/terraform-aws-vault/tree/master/CONTRIBUTING.md) for instructions.



## How is this Module versioned?

This Module follows the principles of [Semantic Versioning](http://semver.org/). You can find each new release,
along with the changelog, in the [Releases Page](../../releases).

During initial development, the major version will be 0 (e.g., `0.x.y`), which indicates the code does not yet have a
stable API. Once we hit `1.0.0`, we will make every effort to maintain a backwards compatible API and use the MAJOR,
MINOR, and PATCH versions on each release to indicate any incompatibilities.



## License

This code is released under the Apache 2.0 License. Please see [LICENSE](https://github.com/hashicorp/terraform-aws-vault/tree/master/LICENSE) and [NOTICE](https://github.com/hashicorp/terraform-aws-vault/tree/master/NOTICE) for more
details.

Copyright &copy; 2020 Gruntwork, Inc.
