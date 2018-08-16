---
layout: "heroku"
page_title: "Heroku: heroku_app_config_vars"
sidebar_current: "docs-heroku-datasource-heroku_app_config_vars-x"
description: |-
  Get latest config vars of a given heroku_app resource.
---

# Data Source: heroku_app_config_vars

Use this data source to get latest config_vars from a heroku app. This allows pull off config_vars addons attach to an app, or pulling off latest current config_var values without relying on remote state of last refresh of `all_config_vars` attribute on a given `heroku_app` resource.

## Example Usage

```hcl
# Look up a Heroku Private Space's peering info. 
data "heroku_app_config_vars" "foo_configs" {
  app   = "${heroku_app.foo.name}"
}

# Initiate a VPC peering connection request.
resource "heroku_app" "foo" {
    name         = "foo"
    organization = "foobars"
    region       = "virginia"
}
```

## Argument Reference

The following arguments are supported:

* `app` - The name of the heroku app to pull config_vars off of
* `depends` - List of strings that are ignored, but can be used to force dependency resolution, if say, you'd like to wait until an addon is provisioned before referencing this resource on another downstream app. This is working around a terraform issue with data sources and the `depends_on` attribute[hashicorp/terraform/issues/11806](https://github.com/hashicorp/terraform/issues/11806)

## Attributes Reference

The following attributes are exported:

* `all_config_vars` - Map of config keys to values pulled off `app`
