# Dnsmasq Install Script

This folder contains a script for installing [Dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html) and configuring 
it to forward requests for a specific domain to Consul. This way, you can easily use Consul as your DNS server for
domain names such as `vault.service.consul`. 

This script has been tested on the following operating systems:

* Ubuntu 16.04
* Amazon Linux

There is a good chance it will work on other flavors of Debian, CentOS, and RHEL as well.



## Quick start

To install Dnsmasq, use `git` to clone this repository at a specific tag (see the [releases page](../../../../releases) 
for all available tags) and run the `install-dnsmasq` script:

```
git clone --branch <VERSION> https://github.com/gruntwork-io/vault-aws-blueprint.git
vault-aws-blueprint/modules/install-dnsmasq/install-dnsmasq --version 2.75-1
```

Note: by default, the `install-dnsmasq` script assumes that a Consul agent is already running locally and connected to 
a Consul cluster (see the [Consul AWS Blueprint](https://github.com/gruntwork-io/consul-aws-blueprint)). After the
install completes, restart `dnsmasq` (e.g. `/etc/init.d/dnsmasq restart`) and queries to the `.consul` domain will be 
resolved via Consul:

```
dig vault.service.consul
```

We recommend running the `install-dnsmasq` script as part of a [Packer](https://www.packer.io/) template to create an
[Amazon Machine Image (AMI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) (see the 
[vault-ami example](/examples/vault-ami) for sample code). 




## Command line Arguments

The `install-dnsmasq` script accepts the following arguments:

* `version VERSION`: Install Dnsmasq version VERSION. Required. 
* `consul-domain DOMAIN`: The domain name to point to Consul. Optional. Default: `consul`.
* `consul-ip IP`: The IP address to use for Consul. Optional. Default: `127.0.0.1`. This assumes a Consul agent is 
  running locally and connected to a Consul cluster.
* `consul-dns-port PORT`: The port Consul uses for DNS requests. Optional. Default: `8600`.

Example:

```
install-dnsmasq --version 2.75-1
```
