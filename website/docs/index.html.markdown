---
layout: "heroku"
page_title: "Provider: Heroku"
sidebar_current: "docs-heroku-index"
description: |-
  The Heroku provider is used to interact with the resources supported by Heroku. The provider needs to be configured with the proper credentials before it can be used.
---

# Heroku Provider

The Heroku provider is used to interact with the resources provided by
Heroku. The provider needs to be configured with the proper credentials
before it can be used.

Use the navigation to the left to read about the available resources.

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

You can provide your credentials staticly by adding `email` and `api_key`
arguments to the Heroku provider block:

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
export HEROKU_EMAIL="ops@company.com"
export HEROKU_API_KEY="heroku_api_key"
terraform plan
```

### Netrc

You can provide your credentials via a `.netrc` file in your home directory.
This file should be in the following format:

```netrc
machine api.heroku.com
  login <your_heroku_email>
  password <your_heroku_api_key>
```

For more information about netrc, please refer to [https://ec.haxx.se/usingcurl-netrc.html](https://ec.haxx.se/usingcurl-netrc.html) 

## Argument Reference

The following arguments are supported:

- `api_key` - (Required) Heroku API token. It must be provided, but it can also
  be sourced from the `HEROKU_API_KEY` environment variable.

- `email` - (Required) Email to be notified by Heroku. It must be provided, but
  it can also be sourced from the `HEROKU_EMAIL` environment variable.

- `headers` - (Optional) Additional Headers to be sent to Heroku. If not provided,
  it can also be sourced from the `HEROKU_HEADERS` environment variable.
