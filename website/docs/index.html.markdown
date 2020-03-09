---
layout: "heroku"
page_title: "Provider: Heroku"
sidebar_current: "docs-heroku-index"
description: |-
  The Heroku provider is used to interact with the resources provided by the Heroku Platform API.
---

# Heroku Provider

The Heroku provider is used to interact with the resources provided by Heroku
Platform API and needs to be configured with credentials before it can be used.

## Background

[Heroku](https://www.heroku.com) is a fully-managed platform that gives you the
simplest path to delivering apps quickly:

* [Using Terraform with Heroku](https://devcenter.heroku.com/articles/using-terraform-with-heroku)
* [Platform API reference](https://devcenter.heroku.com/articles/platform-api-reference)
* [Command Line Interface (CLI)](https://devcenter.heroku.com/articles/heroku-cli)

## Contributing

Development happens in the [GitHub repo](https://github.com/terraform-providers/terraform-provider-heroku):

* [Releases](https://github.com/terraform-providers/terraform-provider-heroku/releases)
* [Changelog](https://github.com/terraform-providers/terraform-provider-heroku/blob/master/CHANGELOG.md)
* [Issues](https://github.com/terraform-providers/terraform-provider-heroku/issues)

## Example Usage

```hcl
# Configure the Heroku provider
provider "heroku" {
  email   = "ops@company.com"
  api_key = var.heroku_api_key
}

# Create a new application
resource "heroku_app" "default" {
  # ...
}
```

## Authentication

The Heroku provider offers a flexible means of providing credentials for
authentication. The following methods are supported, listed in order of
precedence, and explained below:

* Static credentials
* Environment variables
* Netrc

### Static credentials

Credentials can be provided statically by adding `email` and `api_key` arguments
to the Heroku provider block:

```hcl
provider "heroku" {
  email   = "ops@company.com"
  api_key = var.heroku_api_key
}
```

### Environment variables

When the Heroku provider block does not contain an `email` or `api_key`
argument, the missing credentials will be sourced from the environment via the
`HEROKU_EMAIL` and `HEROKU_API_KEY` environment variables respectively:

```hcl
provider "heroku" {}
```

```shell
$ export HEROKU_EMAIL="ops@company.com"
$ export HEROKU_API_KEY="heroku_api_key"
$ terraform plan
Refreshing Terraform state in-memory prior to plan...
```

### Netrc

Credentials can instead be sourced from the [`.netrc`](https://ec.haxx.se/usingcurl-netrc.html)
file in your home directory:

```hcl
provider "heroku" {}
```

```shell
$ cat ~/.netrc
...
machine api.heroku.com
  login <your_heroku_email>
  password <your_heroku_api_key>
...
```

## Argument Reference

The following arguments are supported:

* `api_key` - (Required) Heroku API token. It must be provided, but it can also
  be sourced from [other locations](#Authentication).

* `email` - (Required) Email to be notified by Heroku. It must be provided, but
  it can also be sourced from [other locations](#Authentication).

* `headers` - (Optional) Additional Headers to be sent to Heroku. If not
  provided, it will be sourced from the `HEROKU_HEADERS` environment variable
  (if set).

* `delays` - (Optional) Delays help mitigate issues that can arise due to
  Heroku's eventually consistent data model. Only a single `delays` block may be
  specified and it supports the following arguments:

  * `post_app_create_delay` - (Optional) The number of seconds to wait after an
    app is created. Default is to wait 5 seconds.

  * `post_space_create_delay` - (Optional) The number of seconds to wait after a
    private space is created. Default is to wait 5 seconds.

  * `post_domain_create_delay` - (Optional) The number of seconds to wait after
    a domain is created. Default is to wait 5 seconds.
