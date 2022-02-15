---
layout: "heroku"
page_title: "Heroku: heroku_addon"
sidebar_current: "docs-heroku-datasource-addon-x"
description: |-
  Get information on a Heroku Addon.
---

# Data Source: heroku_addon

Use this data source to get information about a Heroku Addon.

## Example Usage

```hcl-terraform
# Lookup an existing Heroku add-on
data "heroku_addon" "example" {
  name = "example-addon-name"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The add-on name

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the add-on
* `name` - The add-on name
* `app_id` - Heroku app ID
* `plan` - The plan name
* `provider_id` - The ID of the plan provider
* `config_vars` - The Configuration variables of the add-on
