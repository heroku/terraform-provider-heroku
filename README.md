[![Build Status](https://travis-ci.org/terraform-providers/terraform-provider-heroku.svg?branch=master)](https://travis-ci.org/terraform-providers/terraform-provider-heroku)

Terraform Provider
==================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)
- Documentation: https://www.terraform.io/docs/providers/heroku/index.html

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.10.x
-	[Go](https://golang.org/doc/install) 1.8 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-heroku`

```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers; cd $GOPATH/src/github.com/terraform-providers
$ git clone git@github.com:terraform-providers/terraform-provider-heroku
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-heroku
$ make build
```

Using the provider
----------------------
## Fill in for each provider

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.8+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-heroku
...
```

Please see the [TESTING](TESTING.md) guide for detailed instructions on running tests.

### Updating or adding dependencies

This project uses [dep](https://golang.github.io/dep/) for dependency management.

Use the [`dep ensure` command](https://golang.github.io/dep/docs/daily-dep.html) to add, update, & remove dependencies.
