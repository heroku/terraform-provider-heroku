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
data "heroku_addon" "from_another_app" {
  name = "addon-from-another-app"
}

output "heroku_addon_data_basic" {
  value = [
    "Addon from another app",
    "id: ${data.heroku_addon.from_another_app.id}",
    "name: ${data.heroku_addon.from_another_app.name}",
    "app: ${data.heroku_addon.from_another_app.app}",
    "plan: ${data.heroku_addon.from_another_app.plan}",
    "provider_id: ${data.heroku_addon.from_another_app.provider_id}",
    "config_vars: ${join(", ", data.heroku_addon.from_another_app.config_vars)}",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The add-on name

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the add-on
* `name` - The add-on name
* `plan` - The plan name
* `provider_id` - The ID of the plan provider
* `config_vars` - The Configuration variables of the add-on
