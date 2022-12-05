---
layout: "heroku"
page_title: "Heroku: heroku_addon"
sidebar_current: "docs-heroku-resource-addon"
description: |-
  Provides a Heroku Add-On resource. These can be attach services to a Heroku app.
---

# heroku\_addon

Provides a Heroku Add-On resource. These can be attach
services to a Heroku app.

## Example Usage

```hcl-terraform
# Create a new Heroku app
resource "heroku_app" "default" {
  name = "test-app"
}

# Create a database, and configure the app to use it
resource "heroku_addon" "database" {
  app_id = heroku_app.default.id
  plan   = "heroku-postgresql:mini"
}

# Add a web-hook addon for the app
resource "heroku_addon" "webhook" {
  app_id = heroku_app.default.id
  plan   = "deployhooks:http"

  config = {
    url = "http://google.com"
  }
}
```

## Argument Reference

The following arguments are supported:

* `app_id` - (Required) Heroku app ID (do not use app name)
* `plan` - (Required) The addon to add.
* `config` - (Optional) Optional plan configuration.
* `name` - (Optional) Globally unique name of the add-on.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the add-on
* `name` - The add-on name
* `plan` - The plan name
* `provider_id` - The ID of the plan provider
* `config_vars` - The Configuration variables of the add-on
* `config_var_values` - A sensitive map of the add-on's configuration variables. Upon add-on creation, these values will be up-to-date, while the app's own `config_vars` require another Terraform refresh cycle to be updated. Useful when an output contains an add-on config var value, or when a configuration needs to operate on a new add-on during an apply.

## Import

Addons can be imported using the Addon `id`, e.g.

```
$ terraform import heroku_addon.foobar 12345678
```
