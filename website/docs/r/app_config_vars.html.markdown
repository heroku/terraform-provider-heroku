---
layout: "heroku"
page_title: "Heroku: heroku_app_config_vars"
sidebar_current: "docs-heroku-resource-app-config-vars"
description: |-
  Provides a Heroku App Config Variables resource. This can be used to create and manage configs variables for a heroku app.
---

# heroku\_app\_config\_vars

Provides a Heroku App Config Variables resource. This can be used to create and manage configs variables for a heroku app.

## Example Usage

```hcl
resource "heroku_app_config_vars" "foobar" {
  app = "${heroku_app.foobar.name}"

  public {
    name = "RAILS_ENV"
    value = "staging"
  }

  private {
    name = "DATABASE_URL"
    value = "postgres://user:password@some-host.com:1234"
  }
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The Heroku app to link to.
* `public` - (Optional) Configuration variables for the application.
    The config variables in this map are not the final set of configuration
    variables, but rather variables you want present. That is, other
    configuration variables set externally won't be removed by Terraform
    if they aren't present in this list.
* `private` - (Optional) Same as the `public` argument;
    however these config variables will not get displayed in logs or regular output.
    It is recommended one add passwords, tokens or other secret fields to this argument.

## Migrating from config vars defined under the `heroku_app` resource:
TODO: FINISH ME

