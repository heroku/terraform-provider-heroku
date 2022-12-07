Terraform Provider Heroku
=========================

[![Acceptance tests](https://github.com/heroku/terraform-provider-heroku/workflows/Acceptance/badge.svg)](https://github.com/heroku/terraform-provider-heroku/actions?query=workflow%3AAcceptance)

This provider is used to configure resources supported by the [Heroku Platform API](https://devcenter.heroku.com/articles/platform-api-reference).

See the [official documentation](https://www.terraform.io/docs/providers/heroku/index.html) to use this provider in a Terraform configuration.

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/public/img/logo-hashicorp.svg" width="600px">

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 1.x
- [Go](https://golang.org/doc/install) 1.19 (to build the provider plugin)

Usage
-----

The configuration should specify version 5:

```hcl-terraform
provider "heroku" {
  version = "~> 5.0"
}
```

See [Upgrading guide](docs/guides/upgrading.md) for major version migration help.

👓📚 For more usage info, see [Heroku Provider docs](https://www.terraform.io/docs/providers/heroku/index.html).

Development
-----------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine
(version 1.16+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH),
as well as adding `$GOPATH/bin` to your `$PATH`.

### Clone the Provider

With Go language, the repository must be cloned to a specific path in `$GOPATH/src` that matches its module import path.

```sh
mkdir -p $GOPATH/src/github.com/heroku
cd $GOPATH/src/github.com/heroku
git clone git@github.com:heroku/terraform-provider-heroku
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

To use the dev provider with local Terraform, configure a dev override pointing to the directory where `terraform-provider-heroku` can be found:

```
cat << TRC >> ~/.terraformrc
provider_installation {
  dev_overrides {
    "heroku/heroku" = "$GOPATH/bin"
  }
  direct {}
}
TRC
```

#### Terraform Versions Prior to 0.13

Copy the freshly built plugin into Terraform's local plugins directory:

```sh
cp $GOPATH/bin/terraform-provider-heroku ~/.terraform.d/plugins/
```

Set the Heroku provider without a version constraint:

```hcl-terraform
provider "heroku" {}
```

Then, initialize Terraform:

```sh
terraform init
```

### Testing

Please see the [TESTING](TESTING.md) guide for detailed instructions on running tests.

### Updating or adding dependencies

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) for dependency management.

Dependencies can be added or updated as follows:

```bash
$ GO111MODULE=on go get github.com/some/module@release-tag
$ GO111MODULE=on go mod tidy
$ GO111MODULE=on go mod vendor
```

This example will fetch a module at the release tag and record it in your project's go.mod and go.sum files. It's a good idea to tidy up afterward and then copy the dependencies into vendor/.

If a module does not have release tags, then `module@master` can be used instead.

#### Removing dependencies

Remove all usage from your codebase and run:

```bash
$ GO111MODULE=on go mod tidy
$ GO111MODULE=on go mod vendor
```
