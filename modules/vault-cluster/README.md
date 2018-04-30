# Vault Cluster

This folder contains a [Terraform](https://www.terraform.io/) module that can be used to deploy a 
[Vault](https://www.vaultproject.io/) cluster in [AWS](https://aws.amazon.com/) on top of an Auto Scaling Group. This 
module is designed to deploy an [Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) 
that had Vault installed via the [install-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault) module in this Module.




## How do you use this module?

This folder defines a [Terraform module](https://www.terraform.io/docs/modules/usage.html), which you can use in your
code by adding a `module` configuration and setting its `source` parameter to URL of this folder:

```hcl
module "vault_cluster" {
  # Use version v0.0.1 of the vault-cluster module
  source = "github.com/hashicorp/terraform-aws-vault//modules/vault-cluster?ref=v0.0.1"

  # Specify the ID of the Vault AMI. You should build this using the scripts in the install-vault module.
  ami_id = "ami-abcd1234"
  
  # Configure and start Vault during boot. 
  user_data = <<-EOF
              #!/bin/bash
              /opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem
              EOF

  # Add tag to each node in the cluster with value set to var.cluster_name
  cluster_tag_key   = "Name"

  # Optionally add extra tags to each node in the cluster
  cluster_extra_tags = [
    {
      key = "Environment"
      value = "Dev"
      propagate_at_launch = true
    },
    {
      key = "Department"
      value = "Ops"
      propagate_at_launch = true
    }
  ]
  
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
  should install Vault in this AMI using the scripts in the [install-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault) module.
  
* `user_data`: Use this parameter to specify a [User 
  Data](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-shell-scripts) script that each
  server will run during boot. This is where you can use the [run-vault script](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/run-vault) to configure and 
  run Vault. The `run-vault` script is one of the scripts installed by the [install-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault) 
  module. 

You can find the other parameters in [vars.tf](vars.tf).

Check out the [vault-cluster-public](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-public) and 
[vault-cluster-private](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) examples for working sample code.





## How do you use the Vault cluster?

To use the Vault cluster, you will typically need to SSH to each of the Vault servers. If you deployed the
[vault-cluster-private](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) or [vault-cluster-public](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-public) 
examples, the [vault-examples-helper.sh script](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-examples-helper/vault-examples-helper.sh) will do the 
tag lookup for you automatically (note, you must have the [AWS CLI](https://aws.amazon.com/cli/) and 
[jq](https://stedolan.github.io/jq/) installed locally):

```
> ../vault-examples-helper/vault-examples-helper.sh

Your Vault servers are running at the following IP addresses:

11.22.33.44
11.22.33.55
11.22.33.66
```

### Initializing the Vault cluster

The very first time you deploy a new Vault cluster, you need to [initialize the 
Vault](https://www.vaultproject.io/intro/getting-started/deploy.html#initializing-the-vault). The easiest way to do 
this is to SSH to one of the servers that has Vault installed and run:

```
vault operator init

Key 1: 427cd2c310be3b84fe69372e683a790e01
Key 2: 0e2b8f3555b42a232f7ace6fe0e68eaf02
Key 3: 37837e5559b322d0585a6e411614695403
Key 4: 8dd72fd7d1af254de5f82d1270fd87ab04
Key 5: b47fdeb7dda82dbe92d88d3c860f605005
Initial Root Token: eaf5cc32-b48f-7785-5c94-90b5ce300e9b

Vault initialized with 5 keys and a key threshold of 3!
```

Vault will print out the [unseal keys](https://www.vaultproject.io/docs/concepts/seal.html) and a [root 
token](https://www.vaultproject.io/docs/concepts/tokens.html#root-tokens). This is the **only time ever** that all of 
this data is known by Vault, so you **MUST** save it in a secure place immediately! Also, this is the only time that 
the unseal keys should ever be so close together. You should distribute each one to a different, trusted administrator
for safe keeping in completely separate secret stores and NEVER store them all in the same place. 

In fact, a better option is to initial Vault with [PGP, GPG, or 
Keybase](https://www.vaultproject.io/docs/concepts/pgp-gpg-keybase.html) so that each unseal key is encrypted with a
different user's public key. That way, no one, not even the operator running the `init` command can see all the keys
in one place:

```
vault operator init -pgp-keys="keybase:jefferai,keybase:vishalnayak,keybase:sethvargo"

Key 1: wcBMA37rwGt6FS1VAQgAk1q8XQh6yc...
Key 2: wcBMA0wwnMXgRzYYAQgAavqbTCxZGD...
Key 3: wcFMA2DjqDb4YhTAARAAeTFyYxPmUd...
...
```

See [Using PGP, GPG, and Keybase](https://www.vaultproject.io/docs/concepts/pgp-gpg-keybase.html) for more info.


### Unsealing the Vault cluster

Now that you have the unseal keys, you can [unseal Vault](https://www.vaultproject.io/docs/concepts/seal.html) by 
having 3 out of the 5 administrators (or whatever your key shard threshold is) do the following:

1. SSH to a Vault server.
1. Run `vault unseal`.
1. Enter the unseal key when prompted.
1. Repeat for each of the other Vault servers.

Once this process is complete, all the Vault servers will be unsealed and you will be able to start reading and writing
secrets.


### Connecting to the Vault cluster to read and write secrets

There are three ways to connect to Vault:

1. [Access Vault from a Vault server](#access-vault-from-a-vault-server)
1. [Access Vault from other servers in the same AWS account](#access-vault-from-other-servers-in-the-same-aws-account)
1. [Access Vault from the public Internet](#access-vault-from-the-public-internet)


#### Access Vault from a Vault server

When you SSH to a Vault server, the Vault client is already configured to talk to the Vault server on localhost, so 
you can directly run Vault commands:

```
vault read secret/foo

Key                 Value
---                 -----
refresh_interval    768h0m0s
value               bar
```


#### Access Vault from other servers in the same AWS account

To access Vault from a different server in the same account, you need to specify the URL of the Vault cluster. You 
could manually look up the Vault cluster's IP address, but since this module uses Consul not only as a [storage 
backend](https://www.vaultproject.io/docs/configuration/storage/consul.html) but also as a way to register [DNS 
entries](https://www.consul.io/docs/guides/forwarding.html), you can access Vault 
using a nice domain name instead, such as `vault.service.consul`.

To set this up, use the [install-dnsmasq 
module](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/install-dnsmasq) on each server that 
needs to access Vault. This allows you to access Vault from your EC2 Instances as follows:

```
vault -address=https://vault.service.consul:8200 read secret/foo

Key                 Value
---                 -----
refresh_interval    768h0m0s
value               bar
```

You can configure the Vault address as an environment variable:

```
export VAULT_ADDR=https://vault.service.consul:8200
```

That way, you don't have to remember to pass the Vault address every time:

```
vault read secret/foo

Key                 Value
---                 -----
refresh_interval    768h0m0s
value               bar
```

Note that if you're using a self-signed TLS cert (e.g. generated from the [private-tls-cert 
module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/private-tls-cert)), you'll need to have the public key of the CA that signed that cert or you'll get 
an "x509: certificate signed by unknown authority" error. You could pass the certificate manually:
 
```
vault read -ca-cert=/opt/vault/tls/ca.crt.pem secret/foo

Key                 Value
---                 -----
refresh_interval    768h0m0s
value               bar
```

However, to avoid having to add the `-ca-cert` argument to every single call, you can use the [update-certificate-store 
module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/update-certificate-store) to configure the server to trust the CA.

Check out the [vault-cluster-private example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) for working sample code.


#### Access Vault from the public Internet

We **strongly** recommend only running Vault in private subnets. That means it is not directly accessible from the 
public Internet, which reduces your surface area to attackers. If you need users to be able to access Vault from 
outside of AWS, we recommend using VPN to connect to AWS. 
 
If VPN is not an option, and Vault must be accessible from the public Internet, you can use the [vault-elb 
module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/vault-elb) to deploy an [Elastic Load Balancer 
(ELB)](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) in your public subnets, and have all your users
access Vault via this ELB:

```
vault -address=https://<ELB_DNS_NAME> read secret/foo
```

Where `ELB_DNS_NAME` is the DNS name for your ELB, such as `vault.example.com`. You can configure the Vault address as 
an environment variable:

```
export VAULT_ADDR=https://vault.example.com
```

That way, you don't have to remember to pass the Vault address every time:

```
vault read secret/foo
```






## What's included in this module?

This module creates the following architecture:

![Vault architecture](https://github.com/hashicorp/terraform-aws-vault/blob/master/_docs/architecture.png?raw=true)

This architecture consists of the following resources:

* [Auto Scaling Group](#auto-scaling-group)
* [Security Group](#security-group)
* [IAM Role and Permissions](#iam-role-and-permissions)
* [S3 bucket](#s3-bucket) (Optional)


### Auto Scaling Group

This module runs Vault on top of an [Auto Scaling Group (ASG)](https://aws.amazon.com/autoscaling/). Typically, you
should run the ASG with 3 or 5 EC2 Instances spread across multiple [Availability 
Zones](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html). Each of the EC2
Instances should be running an AMI that has had Vault installed via the [install-vault](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault)
module. You pass in the ID of the AMI to run using the `ami_id` input parameter.


### Security Group

Each EC2 Instance in the ASG has a Security Group that allows:
 
* All outbound requests
* Inbound requests on Vault's API port (default: port 8200)
* Inbound requests on Vault's cluster port for server-to-server communication (default: port 8201)
* Inbound SSH requests (default: port 22)

The Security Group ID is exported as an output variable if you need to add additional rules. 

Check out the [Security section](#security) for more details. 


### IAM Role and Permissions

Each EC2 Instance in the ASG has an [IAM Role](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html) attached. 
The IAM Role ARN is exported as an output variable so you can add custom permissions. 


### S3 bucket (Optional)

If `configure_s3_backend` is set to `true`, this module will create an [S3 bucket](https://aws.amazon.com/s3/) that Vault
can use as a storage backend. S3 is a good choice for storage because it provides outstanding durability (99.999999999%)
and availability (99.99%).  Unfortunately, S3 cannot be used for Vault High Availability coordination, so this module expects
a separate Consul server cluster to be deployed as a high availability backend.



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
[How do you handle encryption documentation](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/run-vault#how-do-you-handle-encryption).


### Encryption at rest

Vault servers keep everything in memory and does not write any data to the local hard disk. To persist data, Vault
encrypts it, and sends it off to its storage backends, so no matter how the backend stores that data, it is already
encrypted. By default, this Module uses Consul as a storage backend, so if you want an additional layer of 
protection, you can check out the [official Consul encryption docs](https://www.consul.io/docs/agent/encryption.html) 
and the Consul AWS Module [How do you handle encryption 
docs](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/run-consul#how-do-you-handle-encryption)
for more info.

Note that if you want to enable encryption for the root EBS Volume for your Vault Instances (despite the fact that 
Vault itself doesn't write anything to this volume), you need to enable that in your AMI. If you're creating the AMI 
using Packer (e.g. as shown in the [vault-consul-ami example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-consul-ami)), you need to set the [encrypt_boot 
parameter](https://www.packer.io/docs/builders/amazon-ebs.html#encrypt_boot) to `true`.  


### Dedicated instances

If you wish to use dedicated instances, you can set the `tenancy` parameter to `"dedicated"` in this module. 


### Security groups

This module attaches a security group to each EC2 Instance that allows inbound requests as follows:

* **Vault**: For the Vault API port (default: 8200), you can use the `allowed_inbound_cidr_blocks` parameter to control 
  the list of [CIDR blocks](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) that will be allowed access
  and the `allowed_inbound_security_group_ids` parameter to control the security groups that will be allowed access.  

* **SSH**: For the SSH port (default: 22), you can use the `allowed_ssh_cidr_blocks` parameter to control the list of   
  [CIDR blocks](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) that will be allowed access. You can use the `allowed_ssh_security_group_ids` parameter to control the list of source Security Groups that will be allowed access.
  
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

This module configures Vault to use Consul as a high availability storage backend. This module assumes you already
have Consul servers deployed in a separate cluster. We do not recommend co-locating Vault and Consul servers in the
same cluster because:

1. Vault is a tool built specifically for security, and running any other software on the same server increases its
   surface area to attackers.
1. This Vault Module uses Consul as a high availability storage backend and both Vault and Consul keep their working
   set in memory. That means for every 1 byte of data in Vault, you'd also have 1 byte of data in Consul, doubling
   your memory consumption on each server.

Check out the [Consul AWS Module](https://github.com/hashicorp/terraform-aws-consul) for how to deploy a Consul 
server cluster in AWS. See the [vault-cluster-public](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-public) and 
[vault-cluster-private](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) examples for sample code that shows how to run both a
Vault server cluster and Consul server cluster.


### Monitoring, alerting, log aggregation

This module does not include anything for monitoring, alerting, or log aggregation. All ASGs and EC2 Instances come 
with limited [CloudWatch](https://aws.amazon.com/cloudwatch/) metrics built-in, but beyond that, you will have to 
provide your own solutions. We especially recommend looking into Vault's [Audit 
backends](https://www.vaultproject.io/docs/audit/index.html) for how you can capture detailed logging and audit 
information.

Given that any time Vault crashes, reboots, or restarts, you have to have the Vault admins manually unseal it (see
[What happens if a node crashes?](#what-happens-if-a_node-crashes)), we **strongly** recommend configuring alerts that
notify these admins whenever they need to take action!


### VPCs, subnets, route tables

This module assumes you've already created your network topology (VPC, subnets, route tables, etc). You will need to 
pass in the the relevant info about your network topology (e.g. `vpc_id`, `subnet_ids`) as input variables to this 
module.

