# Vault Cluster

This folder contains a [Terraform](https://www.terraform.io/) module that can be used to deploy a 
[Vault](https://www.vaultproject.io/) cluster in [AWS](https://aws.amazon.com/) on top of an Auto Scaling Group. This 
module is designed to deploy an [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) 
that had Vault installed via the [install-vault](/modules/install-vault) module in this Blueprint.




## How do you use this module?

This folder defines a [Terraform module](https://www.terraform.io/docs/modules/usage.html), which you can use in your
code by adding a `module` configuration and setting its `source` parameter to URL of this folder:

```hcl
module "vault_cluster" {
  # TODO: update this to the final URL
  # Use version v0.0.1 of the vault-cluster module
  source = "github.com/gruntwork-io/vault-aws-blueprint//modules/vault-cluster?ref=v0.0.1"

  # Specify the ID of the Vault AMI. You should build this using the scripts in the install-vault module.
  ami_id = "ami-abcd1234"
  
  # Configure and start Vault during boot. 
  user_data = <<-EOF
              #!/bin/bash
              /opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt --tls-key-file /opt/vault/tls/vault.key
              EOF
  
  # ... See vars.tf for the other parameters you must define for the vault-cluster module
}
```

Note the following parameters:

* `source`: Use this parameter to specify the URL of the vault-cluster module. The double slash (`//`) is intentional 
  and required. Terraform uses it to specify subfolders within a Git repo (see [module 
  sources](https://www.terraform.io/docs/modules/sources.html)). The `ref` parameter specifies a specific Git tag in 
  this repo. That way, instead of using the latest version of this module from the `master` branch, which 
  will change every time you run Terraform, you're using a fixed version of the repo.

* `ami_id`: Use this parameter to specify the ID of a Vault [Amazon Machine Image 
  (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) to deploy on each server in the cluster. You
  should install Vault in this AMI using the scripts in the [install-vault](/modules/install-vault) module.
  
* `user_data`: Use this parameter to specify a [User 
  Data](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-shell-scripts) script that each
  server will run during boot. This is where you can use the [run-vault script](/modules/run-vault) to configure and 
  run Vault. The `run-vault` script is one of the scripts installed by the [install-vault](/modules/install-vault) 
  module. 

You can find the other parameters in [vars.tf](vars.tf).

Check out the [vault-cluster-public](/examples/vault-cluster-public) and 
[vault-cluster-private](/examples/vault-cluster-private) examples for working sample code.





## How do you connect to the Vault cluster?

There are two ways to connect to Vault:

1. [Access Vault from other servers in the same AWS account](#access-vault-from-other-servers-in-the-same-aws-account)
1. [Access Vault from the public Internet](#access-vault-from-the-public-internet)


### Access Vault from other servers in the same AWS account

This module uses Consul not only as a [storage backend](https://www.vaultproject.io/docs/configuration/storage/consul.html)
but also as a way to register [DNS entries](https://www.consul.io/docs/guides/forwarding.html). This allows servers in
the same AWS account to access Vault using DNS (e.g. using an address like `vault.service.consul`).

To set this up, install the [vault-dnsmasq module](/modules/vault-dnsmasq) on each server that needs to access Vault.


### Access Vault from the public Internet

We **strongly** recommend only running Vault in private subnets. That means it is not directly accessible from the 
public Internet, which reduces your surface area to attackers. If you need users to be able to access Vault, we 
recommend using VPN to access Vault. 
 
If VPN is not an option, and Vault must be accessible from the public Internet, you can use the [vault-elb 
module](/modules/vault-elb) to deploy an [Elastic Load Balancer 
(ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) in public subnets, and have all your users
access Vault via this ELB.







## What's included in this module?

This module creates the following architecture:

![Vault architecture](/_docs/architecture.png)

This architecture consists of the following resources:

* [Auto Scaling Group](#auto-scaling-group)
* [Security Group](#security-group)
* [IAM Role and Permissions](#iam-role-and-permissions)


### Auto Scaling Group

This module runs Vault on top of an [Auto Scaling Group (ASG)](https://aws.amazon.com/autoscaling/). Typically, you
should run the ASG with 3 or 5 EC2 Instances spread across multiple [Availability 
Zones](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html). Each of the EC2
Instances should be running an AMI that has had Vault installed via the [install-vault](/modules/install-vault)
module. You pass in the ID of the AMI to run using the `ami_id` input parameter.


### Security Group

Each EC2 Instance in the ASG has a Security Group that allows:
 
* All outbound requests
* Inbound requests on Vault's API port (default: port 8200)
* Inbound SSH requests (default: port 8200)

The Security Group ID is exported as an output variable if you need to add additional rules. 

Check out the [Security section](#security) for more details. 


### IAM Role and Permissions

Each EC2 Instance in the ASG has an [IAM Role](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html) attached. 
The IAM Role ARN is exported as an output variable so you can add permissions. 





## How do you roll out updates?

Please note that Vault does not support true zero-downtime upgrades, but with proper upgrade procedure the downtime 
should be very short (a few hundred milliseconds to a second depending on how the speed of access to the storage 
backend). See the [Vault upgrade guide instructions](https://www.vaultproject.io/docs/guides/upgrading/index.html) for
details.

If you want to deploy a new version of Vault across a cluster deployed with this module, the best way to do that is to:

1. Build a new AMI.
1. Set the `ami_id` parameter to the ID of the new AMI.
1. Run `terraform apply`.

This updates the Launch Configuration of the ASG, so any new Instances in the ASG will have your new AMI, but it does
NOT actually deploy those new instances. To make that happen, you need to:

1. [Replace the standby nodes](#replace-the-standby-nodes)
1. [Replace the primary node](#replace-the-primary-node)


### Replace the standby nodes

For each of the standby nodes:

1. SSH to the EC2 Instance where the Vault standby is running.
1. Execute `sudo supervisorctl stop vault` to have Vault shut down gracefully.
1. Terminate the EC2 Instance.
1. After a minute or two, the ASG should automatically launch a new Instance, with the new AMI, to replace the old one.
1. Have each Vault admin SSH to the new EC2 Instance and unseal it.


### Replace the primary node

The procedure for the primary node is the same, but should be done LAST, after all the standbys have already been
upgraded:

1. SSH to the EC2 Instance where the Vault primary is running. This should be the last server that has the old version
   of your AMI.
1. Execute `sudo supervisorctl stop vault` to have Vault shut down gracefully.
1. Terminate the EC2 Instance.
1. After a minute or two, the ASG should automatically launch a new Instance, with the new AMI, to replace the old one.
1. Have each Vault admin SSH to the new EC2 Instance and unseal it.





## What happens if a node crashes?

There are two ways a Vault node may go down:
 
1. The Vault process may crash. In that case, `supervisor` should restart it automatically. At this point, you will
   need to have each Vault admin SSH to the Instance to unseal it again.
1. The EC2 Instance running Vault dies. In that case, the Auto Scaling Group should launch a replacement automatically. 
   Once again, the Vault admins will have to SSH to the replacement Instance and unseal it.

Given the need for manual intervention, you will want to have alarms set up that go off any time a Vault node gets
restarted.




## Security

Here are some of the main security considerations to keep in mind when using this module:

1. [Encryption in transit](#encryption-in-transit)
1. [Encryption at rest](#encryption-at-rest)
1. [Dedicated instances](#dedicated-instances)
1. [Security groups](#security-groups)
1. [SSH access](#ssh-access)


### Encryption in transit

Vault uses TLS to encrypt its network traffic. For instructions on configuring TLS, have a look at the
[How do you handle encryption documentation](/modules/run-vault#how-do-you-handle-encryption).


### Encryption at rest

Vault servers keep everything in memory and does not write any data to the local hard disk. To persist data, Vault
encrypts it, and sends it off to its storage backends, so no matter how the backend stores that data, it is already
encrypted. By default, this Blueprint uses Consul as a storage backend, so if you want an additional layer of 
protection, you can check out the [official Consul encryption docs](https://www.consul.io/docs/agent/encryption.html) 
and the Consul AWS Blueprint [How do you handle encryption 
docs](https://github.com/gruntwork-io/consul-aws-blueprint/tree/master/modules/run-consul#how-do-you-handle-encryption)
for more info.

Note that if you want to enable encryption for the root EBS Volume for your Vault Instances (despite the fact that 
Vault itself doesn't write anything to this volume), you need to enable that in your AMI. If you're creating the AMI 
using Packer (e.g. as shown in the [vault-ami example](/examples/vault-ami)), you need to set the [encrypt_boot 
parameter](https://www.packer.io/docs/builders/amazon-ebs.html#encrypt_boot) to `true`.  


### Dedicated instances

If you wish to use dedicated instances, you can set the `tenancy` parameter to `"dedicated"` in this module. 


### Security groups

This module attaches a security group to each EC2 Instance that allows inbound requests as follows:

* **Vault**: For the Vault API port (default: 8200), you can use the `allowed_inbound_cidr_blocks` parameter to control 
  the list of [CIDR blocks](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) that will be allowed access.  

* **SSH**: For the SSH port (default: 22), you can use the `allowed_ssh_cidr_blocks` parameter to control the list of   
  [CIDR blocks](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) that will be allowed access. 
  
Note that all the ports mentioned above are configurable via the `xxx_port` variables (e.g. `api_port`). See
[vars.tf](vars.tf) for the full list.  
  
  

### SSH access

You can associate an [EC2 Key Pair](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html) with each
of the EC2 Instances in this cluster by specifying the Key Pair's name in the `ssh_key_name` variable. If you don't
want to associate a Key Pair with these servers, set `ssh_key_name` to an empty string.





## What's NOT included in this module?

This module does NOT handle the following items, which you may want to provide on your own:

* [Consul](#consul)
* [Monitoring, alerting, log aggregation](#monitoring-alerting-log-aggregation)
* [VPCs, subnets, route tables](#vpcs-subnets-route-tables)


### Consul

This module assumes you already have Consul deployed in a separate cluster. We do not recommend co-locating Vault and
Consul in the same cluster because:

1. Vault is a tool built specifically for security, and running any other software on the same server increases its
   surface area to attackers.
1. This Vault Blueprint uses Consul as a storage backend and both Vault and Consul keep their working set in memory. 
   That means for every 1 byte of data in Vault, you'd also have 1 byte of data in Consul, doubling your memory
   consumption on the server.

Check out the [Consul AWS Blueprint](https://github.com/gruntwork-io/consul-aws-blueprint) for how to deploy a Consul 
cluster in AWS. See the [vault-cluster-public](/examples/vault-cluster-public) and 
[vault-cluster-private](/examples/vault-cluster-private) examples for sample code that shows how to run both a
Vault and Consul cluster.


### Monitoring, alerting, log aggregation

This module does not include anything for monitoring, alerting, or log aggregation. All ASGs and EC2 Instances come 
with limited [CloudWatch](https://aws.amazon.com/cloudwatch/) metrics built-in, but beyond that, you will have to 
provide your own solutions. 

Given that any time Vault crashes, reboots, or restarts, you have to have the Vault admins manually unseal it (see
[What happens if a node crashes?](#what-happens-if-a_node-crashes)), we **strongly** recommend configuring alerts that
notify these admins whenever they need to take action!


### VPCs, subnets, route tables

This module assumes you've already created your network topology (VPC, subnets, route tables, etc). You will need to 
pass in the the relevant info about your network topology (e.g. `vpc_id`, `subnet_ids`) as input variables to this 
module.

