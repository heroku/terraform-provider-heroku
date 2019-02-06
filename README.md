[![Build Status](https://travis-ci.org/terraform-providers/terraform-provider-heroku.svg?branch=master)](https://travis-ci.org/terraform-providers/terraform-provider-heroku)

Terraform Provider Heroku
=========================

This provider is used to configure resources supported by the [Heroku Platform API](https://devcenter.heroku.com/articles/platform-api-reference).

See the [official documentation](https://www.terraform.io/docs/providers/heroku/index.html) to use this provider in a Terraform configuration.

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.11.x
-	[Go](https://golang.org/doc/install) 1.11 (to build the provider plugin)

Development
-----------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.11+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

### Clone the Provider

With Go language, the repository must be cloned to a specific path in `$GOPATH/src` that matches its module import path.

```sh
mkdir -p $GOPATH/src/github.com/terraform-providers
cd $GOPATH/src/github.com/terraform-providers
git clone git@github.com:terraform-providers/terraform-provider-heroku
```

### Build the Provider

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-heroku
...
```

### Using the Provider

To use the dev provider with local Terraform, copy the freshly built plugin into Terraform's local plugins directory:

```sh
cp $GOPATH/bin/terraform-provider-heroku ~/.terraform.d/plugins/
```

Set the Heroku provider without a version constraint:

```hcl
provider "heroku" {}
```

Then, initialize Terraform:

```sh
terraform init
```

### Testing

Please see the [TESTING](TESTING.md) guide for detailed instructions on running tests.

### Updating or adding dependencies

This project uses [dep](https://golang.github.io/dep/) for dependency management.

Use the [`dep ensure` command](https://golang.github.io/dep/docs/daily-dep.html) to add, update, & remove dependencies.
