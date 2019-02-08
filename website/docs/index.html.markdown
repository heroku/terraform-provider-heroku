---
layout: "heroku"
page_title: "Provider: Heroku"
sidebar_current: "docs-heroku-index"
description: |-
  The Heroku provider is used to interact with the resources supported by the Heroku Platform API.
---

# Heroku Provider

This provider is used to interact with the resources supported by the Heroku Platform API. The provider needs to be configured with the proper credentials before it can be used.

## Background

[Heroku](https://www.heroku.com)'s fully-managed platform gives you the simplest path to delivering apps quickly:

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
  api_key = "${var.heroku_api_key}"
}

# Create a new application
resource "heroku_app" "default" {
  # ...
}
```

## Authentication

The Heroku provider offers a flexible means of providing credentials for
authentication. The following methods are supported, listed in order
precedence, and explained below:

- Static credentials
- Environment variables
- Netrc

### Static credentials

Static credentials can be provided by adding an `email` and `api_key` in-line
in the Heroku provider block:

```hcl
provider "heroku" {
  email   = "ops@company.com"
  api_key = "${var.heroku_api_key}"
}
```

### Environment variables

You can provide your credentials via the `HEROKU_EMAIL` and `HEROKU_API_KEY`
environment variables, representing your Heroku email address and Heroku api
key, respectively.

```hcl
provider "heroku" {}
```

Usage:

```shell
$ export HEROKU_EMAIL="ops@company.com"
$ export HEROKU_API_KEY="heroku_api_key"
$ terraform plan
```

### Netrc

You can provider your credentials via a `.netrc` file in your home directory.
This file should be in the following format:

```
machine api.heroku.com
  login <your_heroku_email>
  password <your_heroku_api_key>
```

For more information about netrc, please refer to [https://ec.haxx.se/usingcurl-netrc.html](https://ec.haxx.se/usingcurl-netrc.html) 

## Argument Reference

The following arguments are supported:

* `api_key` - (Required) Heroku API token. It must be provided, but it can also
  be sourced from the `HEROKU_API_KEY` environment variable.

* `email` - (Required) Email to be notified by Heroku. It must be provided, but
  it can also be sourced from the `HEROKU_EMAIL` environment variable.

* `headers` - (Optional) Additional Headers to be sent to Heroku. If not provided,
  it can also be sourced from the `HEROKU_HEADERS` environment variable.

* `delays` - (Optional) A `delays` block (documented below). Only one
  `delays` block may be in the configuration. Delays help mitigate issues with 
  eventual consistency in the Heroku back-end service.

The nested `delays` block supports the following:

* `post_app_create_delay` - (Optional) The number of seconds to wait after an app is created. Default is to wait 5 seconds.

* `post_space_create_delay` - (Optional) The number of seconds to wait after a private space is created. Default is to wait 5 seconds.

* `post_domain_create_delay` - (Optional) The number of seconds to wait after a domain is created. Default is to wait 5 seconds.
