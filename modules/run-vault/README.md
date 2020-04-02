# Vault Run Script

This folder contains a script for configuring and running Vault on an [AWS](https://aws.amazon.com/) server. This
script has been tested on the following operating systems:

* Ubuntu 16.04
* Ubuntu 18.04
* Amazon Linux 2

There is a good chance it will work on other flavors of Debian, CentOS, and RHEL as well.




## Quick start

This script assumes you installed it, plus all of its dependencies (including Vault itself), using the [install-vault
module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/install-vault). The default install path is `/opt/vault/bin`, so to start Vault in server mode, you
run:

```
/opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem
```

This will:

1. Generate a Vault configuration file called `default.hcl` in the Vault config dir (default: `/opt/vault/config`).
   See [Vault configuration](#vault-configuration) for details on what this configuration file will contain and how
   to override it with your own configuration.

1. Generate a [systemd](https://www.freedesktop.org/wiki/Software/systemd/) service file called `vault.service` in the systemd
   config dir (default: `/etc/systemd/system`) with a command that will run Vault:
   `vault server -config=/opt/vault/config`.

1. Tell systemd to load the new configuration file, thereby starting Vault.

We recommend using the `run-vault` command as part of [User
Data](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-shell-scripts), so that it executes
when the EC2 Instance is first booting. After running `run-vault` on that initial boot, the `systemd` configuration
will automatically restart Vault if it crashes or the EC2 instance reboots.

Note that `systemd` logs to its own journal by default.  To view the Vault logs, run `journalctl -u vault.service`.  To change
the log output location, you can specify the `StandardOutput` and `StandardError` options by using the `--systemd-stdout` and `--systemd-stderr`
options.  See the [`systemd.exec` man pages](https://www.freedesktop.org/software/systemd/man/systemd.exec.html#StandardOutput=) for available
options, but note that the `file:path` option requires [systemd version >= 236](https://stackoverflow.com/a/48052152), which is not provided
in the base Ubuntu 16.04 and Amazon Linux 2 images.

See the [root example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/root-example) and
[vault-cluster-private](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/vault-cluster-private) examples for fully-working sample code.



## Command line Arguments

The `run-vault` script accepts the following arguments:

Options for Vault Server:

* `--tls-cert-file` (required) Specifies the path to the certificate for TLS. Required. To use a CA certificate, concatenate the primary certificate and the CA certificate together. See [How do you handle encryption?](#how-do-you_handle-encryption) for more info.
* `--tls-key-file` (required) Specifies the path to the private key for the certificate. Required. See [How do you handle encryption?](#how-do-you_handle-encryption) for more info.
* `--port` The port for Vault to listen on. Optional. Default is `8200`.
* `--cluster-port` The port for Vault to listen on for server-to-server requests. Optional. Default is `--port + 1`.
* `--api-addr` The full address to use for [Client Redirection](https://www.vaultproject.io/docs/concepts/ha.html#client-redirection) when running Vault in HA mode. Defaults to "https://[instance_ip]:8200". Optional.
* `--config-dir` The path to the Vault config folder. Optional. Default is the absolute path of `../config`, relative to this script.
* `--bin-dir` The path to the folder with Vault binary. Optional. Default is the absolute path of the parent folder of this script.
* `--log-level` The log verbosity to use with Vault. Optional. Default is `info`.
* `--systemd-stdout` The StandardOutput option of the systemd unit.  Optional.  If not configured, uses systemd's default (journal).
* `--systemd-stderr` The StandardError option of the systemd unit.  Optional.  If not configured, uses systemd's default (inherit).
* `--user` The user to run Vault as. Optional. Default is to use the owner of `--config-dir`.
* `--skip-vault-config` If this flag is set, don't generate a Vault configuration file. Optional. Default is false. This is useful if you have a custom configuration file and don't want to use any of of the default settings from `run-vault`.
* `--enable-s3-backend` If this flag is set, an S3 backend will be enabled in addition to the HA Consul backend. Default is false.
* `--s3-bucket` Specifies the S3 bucket to use to store Vault data. Only used if `--enable-s3-backend` is set.
* `--s3-bucket-path` Specifies the S3 bucket path to use to store Vault data. Only used if `--enable-s3-backend` is set.
* `--s3-bucket-region` Specifies the AWS region where `--s3-bucket` lives. Only used if `--enable-s3-backend` is set.

Options for Vault Agent (`--agent`):

* `--agent` If set, run in Vault Agent mode.  If not set, run as a regular Vault server.  Optional.
* `--agent-vault-address` The hostname or IP address of the Vault server to connect to.  Optional. Default is `vault.service.consul`
* `--agent-vault-port` The port of the Vault server to connect to.  Optional. Default is `8200`
* `--agent-ca-cert-file` Specifies the path to a CA certificate to verify the Vault server's TLS certificate.  Optional.
* `--agent-client-cert-file` Specifies the path to a certificate to use for TLS authentication to the Vault server.  Optional.
* `--agent-client-key-file` Specifies the path to the private key for the client certificate used for TLS authentication to the Vault server.  Optional.
* `--agent-auth-mount-path` The Vault mount path to the auth method used for auto-auth.  Optional.  Defaults to auth/aws
* `--agent-auth-type` The Vault AWS auth type to use for auto-auth.  Required with `--agent`.  Must be either `iam` or `ec2`
* `--agent-auth-role` The Vault role to authenticate against.  Required with `--agent`

Optional Arguments for enabling the [AWS KMS auto-unseal](https://learn.hashicorp.com/vault/operations/ops-autounseal-aws-kms) (Vault Enterprise or 1.0 and above):

* `--enable-auto-unseal` If this flag is set, enable the AWS KMS Auto-unseal feature. Default is false.
* `--auto-unseal-kms-key-id` The key id of the AWS KMS key to be used for encryption and decryption. Required if `--enable-auto-unseal` is enabled.
* `--auto-unseal-kms-key-region` The AWS region where the encryption key lives. Required if `--enable-auto-unseal` is enabled.
* `--auto-unseal-endpoint` The KMS API endpoint to be used to make AWS KMS requests. Optional. Defaults to `""`. Only used if `--enable-auto-unseal` is enabled.

Example:

```
/opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem
```

Or if you want to enable an S3 backend:

```
/opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem --enable-s3-backend --s3-bucket my-vault-bucket --s3-bucket-region us-east-1
```



## Vault configuration

`run-vault` generates a configuration file for Vault called `default.hcl` that tries to figure out reasonable
defaults for a Vault cluster in AWS. Check out the [Vault Configuration Files
documentation](https://www.vaultproject.io/docs/configuration/index.html) for what configuration settings are
available.


### Default configuration

`run-vault` sets the following configuration values by default:
* [ui](https://www.vaultproject.io/docs/configuration/index.html#ui):
      Set to "ui = true" only when the installed vault version is >=0.10.0.

* [api_addr](https://www.vaultproject.io/docs/configuration/index.html#api_addr):
      Set to `https://<PRIVATE_IP>:<PORT>` where `PRIVATE_IP` is the Instance's private IP fetched from
      [Metadata](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html) and `PORT` is
      the value passed to `--port`.
* [cluster_addr](https://www.vaultproject.io/docs/configuration/index.html#cluster_addr):
      Set to `https://<PRIVATE_IP>:<CLUSTER_PORT>` where `PRIVATE_IP` is the Instance's private IP fetched from
      [Metadata](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html) and `CLUSTER_PORT` is
      the value passed to `--cluster-port`.

* [storage](https://www.vaultproject.io/docs/configuration/index.html#storage): Configure Consul as the storage backend
  with the following settings:

    * [address](https://www.vaultproject.io/docs/configuration/storage/consul.html#address): Set the address to
      `127.0.0.1:8500`. This is based on the assumption that the Consul agent is running on the same server.
    * [scheme](https://www.vaultproject.io/docs/configuration/storage/consul.html#scheme): Set to `http` since our
      connection is to a Consul agent running on the same server.
    * [path](https://www.vaultproject.io/docs/configuration/storage/consul.html#path): Set to `vault/`.
    * [service](https://www.vaultproject.io/docs/configuration/storage/consul.html#service): Set to `vault`.


* [listener](https://www.vaultproject.io/docs/configuration/index.html#listener): Configure a [TCP
  listener](https://www.vaultproject.io/docs/configuration/listener/tcp.html) with the following settings:

    * [address](https://www.vaultproject.io/docs/configuration/listener/tcp.html#address): Bind to `0.0.0.0:<PORT>`
      where `PORT` is the value passed to `--port`.
    * [cluster_address](https://www.vaultproject.io/docs/configuration/listener/tcp.html#cluster_address): Bind to
      `0.0.0.0:<CLUSTER_PORT>` where `CLUSTER` is the value passed to `--cluster-port`.
    * [tls_cert_file](https://www.vaultproject.io/docs/configuration/listener/tcp.html#tls_cert_file): Set to the
      `--tls-cert-file` parameter.
    * [tls_key_file](https://www.vaultproject.io/docs/configuration/listener/tcp.html#tls_key_file): Set to the
      `--tls-key-file` parameter.

`run-vault` can optionally set the following configuration values:

* [storage](https://www.vaultproject.io/docs/configuration/index.html#storage): Set the `--enable-s3-backend` flag to
  configure S3 as an additional (non-HA) storage backend with the following settings:

    * [bucket](https://www.vaultproject.io/docs/configuration/storage/s3.html#bucket): Set to the `--s3-bucket`
      parameter.
    * [path](https://www.vaultproject.io/docs/configuration/storage/s3.html#path): Set to the `--s3-bucket-path`
      parameter.
    * [region](https://www.vaultproject.io/docs/configuration/storage/s3.html#region): Set to the `--s3-bucket-region`
      parameter.

### Overriding the configuration

To override the default configuration, simply put your own configuration file in the Vault config folder (default:
`/opt/vault/config`), but with a name that comes later in the alphabet than `default.hcl` (e.g.
`my-custom-config.hcl`). Vault will load all the `.hcl` configuration files in the config dir and merge them together
in alphabetical order, so that settings in files that come later in the alphabet will override the earlier ones.

For example, to set a custom `cluster_name` setting, you could create a file called `name.hcl` with the
contents:

```hcl
cluster_name = "my-custom-name"
```

If you want to override *all* the default settings, you can tell `run-vault` not to generate a default config file
at all using the `--skip-vault-config` flag:

```
/opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem --skip-vault-config
```




## How do you handle encryption?

Vault uses TLS to encrypt all data in transit. To configure encryption, you must do the following:

1. [Provide TLS certificates](#provide-tls-certificates)
1. [Consul encryption](#consul-encryption)


### Provide TLS certificates

When you execute the `run-vault` script, you need to provide the paths to the public and private keys of a TLS
certificate:

```
/opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem
```

See the [private-tls-cert module](https://github.com/hashicorp/terraform-aws-vault/tree/master/modules/private-tls-cert) for information on how to generate a TLS certificate.


### Consul encryption

Since this Vault Module uses Consul as a storage backend (and optionally S3), you may want to enable encryption for your storage too.
Note that Vault encrypts any data *before* sending it to a storage backend, so this isn't strictly necessary, but may be a good
extra layer of security.

By default, the Vault server nodes communicate with a local Consul agent running on the same server over (unencrypted)
HTTP. However, you can configure those agents to talk to the Consul servers using TLS. Check out the [official Consul
encryption docs](https://www.consul.io/docs/agent/encryption.html) and the Consul AWS Module [How do you handle
encryption docs](https://github.com/hashicorp/terraform-aws-consul/tree/master/modules/run-consul#how-do-you-handle-encryption)
for more info.
