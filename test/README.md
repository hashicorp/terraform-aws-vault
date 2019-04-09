# Tests

This folder contains automated tests for this Module. All of the tests are written in [Go](https://golang.org/). 
Most of these are "integration tests" that deploy real infrastructure using Terraform and verify that infrastructure 
works as expected using a helper library called [Terratest](https://github.com/gruntwork-io/terratest).  



## WARNING WARNING WARNING

**Note #1**: Many of these tests create real resources in an AWS account and then try to clean those resources up at 
the end of a test run. That means these tests may cost you money to run! When adding tests, please be considerate of 
the resources you create and take extra care to clean everything up when you're done!

**Note #2**: Never forcefully shut the tests down (e.g. by hitting `CTRL + C`) or the cleanup tasks won't run!

**Note #3**: We set `-timeout 60m` on all tests not because they necessarily take that long, but because Go has a
default test timeout of 10 minutes, after which it forcefully kills the tests with a `SIGQUIT`, preventing the cleanup
tasks from running. Therefore, we set an overlying long timeout to make sure all tests have enough time to finish and 
clean up.



## Running the tests

### Prerequisites

- Install the latest version of [Go](https://golang.org/).
- Install [dep](https://github.com/golang/dep) for Go dependency management.
- Install [Terraform](https://www.terraform.io/downloads.html).
- Configure your AWS credentials using one of the [options supported by the AWS 
  SDK](http://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html). Usually, the easiest option is to
  set the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.


### One-time setup

Download Go dependencies using dep:

```
cd test
dep ensure
```


### Run all the tests

```bash
cd test
go test -v -timeout 60m
```


### Run a specific test

To run a specific test called `TestFoo`:

```bash
cd test
go test -v -timeout 60m -run TestFoo
```

### Special note on the root-example test

As part of the tests for the [root example](https://github.com/hashicorp/terraform-aws-vault/tree/master/examples/root-example), we try to connect to the
Vault cluster via its ELB. If you've configure the test to set up a Route 53 domain name for the ELB, the tests will
try to talk to Vault via this domain name; otherwise, they will talk directly to the ELB's domain name, albeit with
the TLS check disabled, as the TLS cert will not include the ELB's domain name (since that's generated dynamically).

To tell the tests to use a Route 53 domain name for the ELB, specify the domain to use (which must already be 
configured with a Route 53 hosted zone in your AWS account!) using the `VAULT_HOSTED_ZONE_DOMAIN_NAME` environment
variable:

```bash
cd test
export VAULT_HOSTED_ZONE_DOMAIN_NAME="gruntwork.in"
go test -v -timeout 60m
```

  
