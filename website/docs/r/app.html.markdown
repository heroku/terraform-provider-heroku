---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-resource-app-x"
description: |-
  Provides a Heroku App resource. This can be used to create and manage applications on Heroku.
---

# heroku\_app

Provides a Heroku App resource. This can be used to
create and manage applications on Heroku.

## Example Usage

```hcl
# Create a new Heroku app
resource "heroku_app" "default" {
  name   = "my-cool-app"
  region = "us"

  config_vars {
    FOOBAR = "baz"
  }

  buildpacks = [
    "heroku/go"
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application. In Heroku, this is also the
   unique ID, so it must be unique and have a minimum of 3 characters.
* `region` - (Required) The region that the app should be deployed in.
* `stack` - (Optional) The application stack is what platform to run the application
   in.
* `buildpacks` - (Optional) Buildpack names or URLs for the application.
     Buildpacks configured externally won't be altered if this is not present.
* `config_vars` - (Optional) Configuration variables for the application.
     The config variables in this map are not the final set of configuration
     variables, but rather variables you want present. That is, other
     configuration variables set externally won't be removed by Terraform
     if they aren't present in this list.
* `space` - (Optional) The name of a private space to create the app in.
* `internal_routing` - (Optional) If true, the application will be routable
  only internally in a private space. This option is only available for apps
  that also specify `space`. This feature is currently only available in
  private beta. Contact Heroku Support for more details.
* `organization` - (Optional) A block that can be specified once to define
     organization settings for this app. The fields for this block are
     documented below.
* `acm` - (Optional) The flag representing Automated Certificate Management for the app.

The `organization` block supports:

* `name` (string) - The name of the organization.
* `locked` (boolean)
* `personal` (boolean)

~> **NOTE:** Internal routing is currently a beta feature that requires you email Heroku Support to have it enabled. Without this feature enabled by Heroku Support the `internal_routing` attribute will be ignored.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the app. This is also the name of the application.
* `name` - The name of the application. In Heroku, this is also the
   unique ID.
* `stack` - The application stack is what platform to run the application
   in.
* `space` - The private space the app should run in.
* `internal_routing` - Whether internal routing is enabled the private space
  app.
* `region` - The region that the app should be deployed in.
* `git_url` - The Git URL for the application. This is used for
   deploying new versions of the app.
* `web_url` - The web (HTTP) URL that the application can be accessed
   at by default.
* `heroku_hostname` - A hostname for the Heroku application, suitable
   for pointing DNS records.
* `all_config_vars` - A map of all of the configuration variables that
    exist for the app, containing both those set by Terraform and those
    set externally.

## Import

Apps can be imported using the App `id`, e.g.

```
$ terraform import heroku_app.foobar MyApp
```
