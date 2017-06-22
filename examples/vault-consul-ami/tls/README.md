# Example TLS Certificate Files

### Do NOT use these files in production!

In a production setting, your TLS private key represents a critical secret. If it were stolen, its possessor could 
impersonate your Vault server! For that reason, do NOT use these TLS certificate files in a public setting. They are 
here only for convenience when building examples.

### Files

The files in this folder are needed by Vault to accept HTTPS requests. They are:

- **ca.crt.pem**: The public certificate of the Certificate Authority used to create these files.
- **vault.crt.pem:** The TLS public certificate issued by the Certificate Authority of the Vault server.
- **vault.key.pem:** The TLS private key that corresponds to the TLS public certificate.

The TLS files are configured as follows:

- The Vault Server may be reached via TLS at `vault.service.consul`, `vault.example.com`, or `127.0.0.1`.
- The TLS certificate is valid until May 26, 2042.