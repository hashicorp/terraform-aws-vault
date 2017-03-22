# Private TLS Cert

This module can be used to generate the public and private keys of a self-signed TLS certificate. This certificate is 
meant to be used with **private** services, such as a Vault cluster accessed solely within your AWS account. For 
publicly-accessible services, especially services you access through a web browser, you should NOT use 
this module, and instead get certificates from a commercial Certificate Authority, such as [Let's 
Encrypt](https://letsencrypt.org/).




## Quick start

1. Copy this module to your computer.

1. Open `vars.tf` and fill in the variables that do not have a default.

1. DO NOT configure Terraform remote state storage for this code. You do NOT want to store the state files as they 
   will contain the private keys for the certificates.

1. Run `terraform apply`. The output will show you the paths to the generated files:

    ```
    Outputs:
    
    private_key_file_path = vault.key.pem
    public_key_file_path = vault.crt.pem
    ```
    
1. Delete your local Terraform state:

    ```
    rm -rf terraform.tfstate*
    ```

   The Terraform state will contain the private keys for the certificates, so it's important to clean it up!

1. To inspect a certificate, you can use OpenSSL:

    ```
    openssl x509 -inform pem -noout -text -in vault.crt.pem
    ```

Now that you have your TLS certs, check out the next section for how to use them.




## Using TLS certs


### Servers

Distribute the private and public keys (the files at `private_key_file_path` and `public_key_file_path`) to the 
servers that will use them to handle TLS connections (e.g. Vault). For example, to run Vault with the [run-vault 
module](/modules/run-vault), you need to pass it the TLS certs: 

```
/opt/vault/bin/run-vault --tls-cert-file /opt/vault/tls/vault.crt.pem --tls-key-file /opt/vault/tls/vault.key.pem
```   

We strongly recommend encrypting the private key file (e.g. using [KMS](https://aws.amazon.com/kms/)) while it's in 
transit to the servers that will use it.   


### Clients   
   
Distribute the JUST the public key (the file at `public_key_file_path`) to any clients of those services so they can 
validate the server's TLS cert. Without the public key, the clients will reject any TLS connections: 

```
vault read secret/foo

Error initializing Vault: Get https://127.0.0.1:8200/v1/secret/foo: x509: certificate signed by unknown authority
```

Most TLS clients offer a way to explicitly specify extra public keys that you want to trust. For example, with 
Vault, you do this via the `-ca-cert` argument:

```
vault read -ca-cert=vault.crt.pem secret/foo

Key                 Value
---                 -----
refresh_interval    768h0m0s
value               bar
```

As an alternative, you can configure the certificate trust on your server so that all TLS clients trust your 
certificate by running the [update-certificate-store module](/modules/update-certificate-store) on your server. Once 
you do that, your system will trust the public key without having to pass it in explicitly:

```
update-certificate-store --cert-file /opt/vault/tls/vault.crt.pem
vault read secret/foo

Key                 Value
---                 -----
refresh_interval    768h0m0s
value               bar
```




