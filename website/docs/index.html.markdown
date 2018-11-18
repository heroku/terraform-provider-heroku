---
layout: "heroku"
page_title: "Provider: Heroku"
sidebar_current: "docs-heroku-index"
description: |-
  The Heroku provider is used to interact with the resources supported by Heroku. The provider needs to be configured with the proper credentials before it can be used.
---

# Heroku Provider

The Heroku provider is used to interact with the resources supported by
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

<<<<<<< HEAD
Static credentials can be provided by adding an `email` and `api_key` in-line
in the Heroku provider block:
=======
Static credentials can be provided by adding an `email` and `api_key` in-line in the Heroku provider block:
>>>>>>> Fix doc formatting

```hcl
provider "heroku" {
  email   = "ops@company.com"
  api_key = "${var.heroku_api_key}"
}
```

### Environment variables
<<<<<<< HEAD

You can provide your credentials via the `HEROKU_EMAIL` and `HEROKU_API_KEY`
environment variables, representing your Heroku email address and Heroku api
key, respectively.
=======
You can provide your credentials via the `HEROKU_EMAIL` and `HEROKU_API_KEY` environment variables,
representing your Heroku email address and Heroku api key, respectively.
>>>>>>> Fix doc formatting

```hcl
provider "heroku" {}
```

Usage:

<<<<<<< HEAD
```shell
=======
```
>>>>>>> Fix doc formatting
$ export HEROKU_EMAIL="ops@company.com"
$ export HEROKU_API_KEY="heroku_api_key"
$ terraform plan
```

### Netrc
<<<<<<< HEAD

You can provider your credentials via a `.netrc` file in your home directory.
This file should be in the following format:

```
machine api.heroku.com
  login <your_heroku_email>
  password <your_heroku_api_key>
```

For more information about netrc, please refer to [https://ec.haxx.se/usingcurl-netrc.html](https://ec.haxx.se/usingcurl-netrc.html) 
=======
You can provider your credentials via a `.netrc` file in your home directory. This file should be in this format:

 ```
 machine api.heroku.com
   login <your_heroku_email>
   password <your_heroku_api_key>
 ```

 For more information about netrc, please refer to [https://ec.haxx.se/usingcurl-netrc.html](https://ec.haxx.se/usingcurl-netrc.html)
>>>>>>> Fix doc formatting

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

* `post_app_create_delay` - (Optional) The number of seconds to wait after an app is created. Default is to wait 8 seconds.

* `post_space_create_delay` - (Optional) The number of seconds to wait after a private space is created. Default is to wait 8 seconds.

* `post_domain_create_delay` - (Optional) The number of seconds to wait after a domain is created. Default is to wait 8 seconds.