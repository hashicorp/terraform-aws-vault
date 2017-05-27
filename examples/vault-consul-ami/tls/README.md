# Example TLS Certificate Files

The files in this folder are needed by Vault to accept HTTPS requests. They are:

- **ca.crt.pem**: The public certificate of the Certificate Authority used to create these files.
- **vault.crt.pem:** The TLS public certificate issued by the Certificate Authority of the Vault server.
- **vault.key.pem:** The TLS private key that corresponds to the TLS public certificate.

The TLS files are configured as follows:

- The Vault Server may be reached via TLS at `vault.service.consul`, `vault.example.com`, or `127.0.0.1`.
- The TLS certificate is valid until May 26, 2042.

### For Quick Start Purposes Only

These TLS certificate files are here solely to facilitate a quick start with this blueprint. **In a production setting, 
you should generate these files on your own** according to your organization's standards for issueing TLS certificates.  