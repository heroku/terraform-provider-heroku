---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-resource-app-x"
description: |-
  Provides a Heroku App resource. This can be used to create and manage applications on Heroku.
---

# heroku\_app

Provides a Heroku App resource. This can be used to create and manage applications on Heroku.

-> **Always reference apps by ID (UUID) in Terraform configuration**
Starting with v5.0 of this provider, all HCL app references are by ID. Read more details in [Upgrading](guides/upgrading.html).

## Example Usage

```hcl-terraform
resource "heroku_app" "default" {
  name   = "my-cool-app"
  region = "us"

  config_vars = {
    FOOBAR = "baz"
  }

  buildpacks = [
    "heroku/go"
  ]
}
```

## Example Usage for a Team

A Heroku "team" was originally called an "organization", and that is still the identifier used in this resource.

```hcl-terraform
resource "heroku_app" "default" {
  name   = "my-cool-app"
  region = "us"

  organization {
    name = "my-cool-team"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application. In Heroku, this is also the
   unique ID, so it must be unique and have a minimum of 3 characters.
* `region` - (Required) The region that the app should be deployed in.
* `stack` - (Optional) The application stack is what platform to run the application in.
* `buildpacks` - (Optional) Buildpack names or URLs for the application.
  Buildpacks configured externally won't be altered if this is not present.
* `config_vars`<sup>[1](#deleting-vars)</sup> - (Optional) Configuration variables for the application.
     The config variables in this map are not the final set of configuration
     variables, but rather variables you want present. That is, other
     configuration variables set externally won't be removed by Terraform
     if they aren't present in this list.
* `sensitive_config_vars`<sup>[1](#deleting-vars)</sup> - (Optional) This argument is the same as `config_vars`.
     The main difference between the two is when `sensitive_config_vars` outputs
     are displayed on-screen following a terraform apply or terraform refresh,
     they are redacted, with <sensitive> displayed in place of their value.
     It is recommended to put private keys, passwords, etc in this argument.
* `space` - (Optional) The name of a private space to create the app in.
* `internal_routing` - (Optional) If true, the application will be routable
  only internally in a private space. This option is only available for apps
  that also specify `space`.
* `organization` - (Optional) A block that can be specified once to define
     Heroku Team settings for this app. The fields for this block are
     documented below.
* `acm` - (Optional) The flag representing Automated Certificate Management for the app.

The `organization` block supports:
* `name` (string) - The name of the Heroku Team.
* `locked` (boolean) - Are other team members forbidden from joining this app.
* `personal` (boolean) - Force creation of the app in the user account even if a default team is set.

### Deleting vars

Deleting an entire `config_vars` or `sensitive_config_vars` map from a `heroku_app`
configuration will not actually remove the vars on the remote resource. To remove an existing variable,
leave these attribute maps in-place and delete only its entries from the map. Once these attributes are
empty, the map itself may be deleted from the configuration. Otherwise if one deletes the map with existing
entries, the config vars will not be deleted from the remote resource.

This is especially important if you are migrating all `config_vars` to `sensitive_config_vars` or migrating
config vars to `heroku_app_config_association` resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID (UUID) of the app.
* `name` - The name of the app.
* `stack` - The application stack is what platform to run the application in.
* `space` - The private space the app should run in.
* `internal_routing` - Whether internal routing is enabled the private space app.
* `region` - The region that the app should be deployed in.
* `git_url` - The Git URL for the application. This is used for
   deploying new versions of the app.
* `web_url` - The web (HTTP) URL that the application can be accessed
   at by default.
* `heroku_hostname` - A hostname for the Heroku application, suitable
   for pointing DNS records.
* `all_config_vars` - A map of all configuration variables that
  exist for the app, containing both those set by Terraform and those
  set externally. (These are treated as "sensitive" so that
  their values are redacted in console output.) This attribute is not set in state if the `provider`
  attribute `set_app_all_config_vars_in_state` is `false`.
* `uuid` - The unique UUID of the Heroku app. **NOTE:** Use this for `null_resource` triggers.

## Import

Apps can be imported using an existing app's `UUID` or name.

For example:
```
$ terraform import heroku_app.foobar MyApp
$ terraform import heroku_app.foobar e74ac056-7d00-4a7e-aa80-df4bc413a825
```
