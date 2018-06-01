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
$ make bin
...
$ $GOPATH/bin/terraform-provider-heroku
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`. You will need to set at least three environment variables to run them:

* `HEROKU_ORGANIZATION` – The organization to run the tests against.
* `HEROKU_SPACES_ORGANIZATION` – The organization to run the Heroku Private Space tests against. 
* `HEROKU_API_KEY` – A valid API key that has access to the two organizations listed above.

Things to keep in mind when running acceptance tests:

* The tests take roughly 60 minutes to run. Creating Heroku Private Spaces can take 10 minutes and the tests create/destroy a few private spaces.
* *Acceptance tests create real resources, and often cost money to run.*

```sh
$ make testacc
```
