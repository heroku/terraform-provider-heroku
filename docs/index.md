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

## Guides

* [Upgrading](guides/upgrading.html)
* [Secure Practices](guides/security.html)

## Contributing

Development happens in the [GitHub repo](https://github.com/heroku/terraform-provider-heroku):

* [Releases](https://github.com/heroku/terraform-provider-heroku/releases)
* [Changelog](https://github.com/heroku/terraform-provider-heroku/blob/master/CHANGELOG.md)
* [Issues](https://github.com/heroku/terraform-provider-heroku/issues)

## Example Usage

```hcl-terraform
# Configure the Heroku provider
provider "heroku" {
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
* HCP Terraform workload identity

### Generating tokens

All authentication tokens must be generated with one of these methods:

* [Heroku Dashboard](https://dashboard.heroku.com) → Account Settings → Applications → [Authorizations](https://dashboard.heroku.com/account/applications)
* `heroku auth` command of the [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli)
* [Heroku Platform APIs: OAuth](https://devcenter.heroku.com/articles/platform-api-reference#oauth-authorization)

🔐  See [Secure Practices](guides/security.html#authentication) for help creating a safe API token.

⛔️  Direct username-password authentication is [no longer supported by Heroku API](https://devcenter.heroku.com/changelog-items/2516).

### Static credentials

Credentials can be provided statically by adding `api_key` property
to the Heroku provider block:

```hcl-terraform
variable "heroku_api_key" {
  type      = string
  sensitive = true
}

provider "heroku" {
  api_key = var.heroku_api_key
}
```

### Environment variables

When the Heroku provider block does not contain an `api_key`
argument, the missing credentials will be sourced from the environment via the 
`HEROKU_API_KEY` environment variable:

```hcl-terraform
provider "heroku" {}
```

```shell
$ export HEROKU_API_KEY="<heroku_auth_token>"
$ terraform plan
Refreshing Terraform state in-memory prior to plan...
```

### Netrc

Credentials can instead be sourced from the [`.netrc`](https://ec.haxx.se/usingcurl-netrc.html)
file in your home directory:

```hcl-terraform
provider "heroku" {}
```

```shell
$ cat ~/.netrc
...
machine api.heroku.com
  login <ignored, can be any value>
  password <heroku_auth_token>
...
```

The directory containing the `.netrc` file can be overridden by the `NETRC` environment variable as described [here](https://www.gnu.org/software/inetutils/manual/html_node/The-_002enetrc-file.html).

### HCP Terraform workload identity

When running in HCP Terraform (Terraform Cloud), the provider can authenticate
with the workspace's [workload identity token](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/workload-identity-tokens)
instead of a long-lived API key. Set this environment variable in the workspace:

```shell
TFC_WORKLOAD_IDENTITY_AUDIENCE=https://api.heroku.com
```

HCP Terraform then injects `TFC_WORKLOAD_IDENTITY_TOKEN` into each run, and the
provider sends it to the Heroku API as a bearer token. No provider
configuration is needed:

```hcl-terraform
provider "heroku" {}
```

* **Precedence:** `api_key`, `HEROKU_API_KEY` and netrc credentials all win over
  the workload identity token. Remove the shared API key from the workspace to
  switch to workload identity.
* **Opt-out:** set `tfc_workload_identity = false` in the provider block, or
  `HEROKU_TFC_WORKLOAD_IDENTITY=false` in the environment.
* **Token age:** the Heroku API rejects workload identity tokens issued more
  than 30 minutes ago, so a single plan or apply phase must finish within
  that window.
* Tagged tokens (`TFC_WORKLOAD_IDENTITY_TOKEN_<TAG>`) are not supported.
* Self-hosted HCP Terraform agents need v1.7.0 or later.
* The Heroku API must have a platform user configured to trust the workspace.

## Argument Reference

The following arguments are supported:

* `api_key` - (Required) Heroku API token. It must be provided, but it can also
  be sourced from [other locations](#Authentication). See also [Secure Practices](guides/security.html).

* `email` - (Ignored) This field originally supported username-password authentication, 
  but has since [been deprecated](https://devcenter.heroku.com/changelog-items/2516).
  Instead, simply set an auth token in the `api_key` property.

* `tfc_workload_identity` - (Optional) Use the HCP Terraform workload identity token
  (`TFC_WORKLOAD_IDENTITY_TOKEN`) when no API key is configured. Defaults to `true`, or
  the value of the `HEROKU_TFC_WORKLOAD_IDENTITY` environment variable.
  See [HCP Terraform workload identity](#hcp-terraform-workload-identity).

* `headers` - (Optional) Additional Headers to be sent to Heroku, as a string-encoded JSON object, 
  for example: `{"X-Custom-Header":"yes","X-Custom-Header-Too":"no"}`. If not provided, it will be 
  sourced from the `HEROKU_HEADERS` environment variable (if set).

* `customizations` - (Optional) Various attributes altering the behavior of certain resources.
  Only a single `customizations` block may be specified, and it supports the following arguments:

  * `set_app_all_config_vars_in_state` - (Optional) Controls whether the `heroku_app.all_config_vars` attribute
    is set in the state file. Normally a snapshot of all config vars is stored in state, even though they are
    not managed by Terraform, such as secrets set via `heroku config` CLI, web dashboard, or add-ons like 
    Postgres' `DATABASE_URL`. Set to `false` to only track managed config vars in the state. Defaults to `true`.
    See also [Secure Practices](guides/security.html).

  * `set_addon_config_vars_in_state` - (Optional) Controls whether the `heroku_addon.config_var_values` attribute
    is set in the state file. The attribute stores each addon's config vars in Terraform state. This means 
    sensitive add-on config vars, such as Postgres' `DATABASE_URL`, are always accessible in the state.
    Set to `false` to prevent capturing these values. Defaults to `true`.
    See also [Secure Practices](guides/security.html).

* `delays` - (Optional) Delays help mitigate issues that can arise due to
  Heroku's eventually consistent data model. Only a single `delays` block may be
  specified, and it supports the following arguments:

  * `post_app_create_delay` - (Optional) The number of seconds to wait after an
    app is created. Default is to wait 5 seconds.

  * `post_space_create_delay` - (Optional) The number of seconds to wait after a
    private space is created. Default is to wait 5 seconds.

  * `post_domain_create_delay` - (Optional) The number of seconds to wait after
    a domain is created. Default is to wait 5 seconds.

* `timeouts` - (Optional) Define a max duration the provider will wait for certain resources
  to be properly modified before proceeding with further action(s). Only a single `timeouts` block may be specified,
  and it supports the following arguments:

  * `addon_create_timeout` - (Optional) The number of minutes for the provider to wait for an addon to be
  created/provisioned. Defaults to 20 minutes. Minimum required value is 10 minutes.
