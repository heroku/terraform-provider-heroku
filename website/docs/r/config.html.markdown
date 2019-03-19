---
layout: "heroku"
page_title: "Heroku: heroku_config"
sidebar_current: "docs-heroku-resource-config"
description: |-
  Provides a Heroku Config resource, making it possible to define variables that can be used in other Heroku terraform resources.
---

# heroku\_config
Provides a Heroku Config resource, making it possible to define variables 
to be used throughout your Heroku terraform configurations. Combined with `heroku_config_association`,
these two resources enable users to decouple setting config var(s) from the `heroku_app` resource.

~> **NOTE:** Unlike most Terraform resources, this resource **DOES NOT** by itself create, update or delete anything on Heroku. 
A [`heroku_app_config_association`](app_config_association.html), `heroku_app.config_vars`, or `heroku_app.sensitive_config_vars` is required to actually set these values on Heroku apps.

## Example HCL
```hcl
resource "heroku_config" "endpoints" {
    name = "endpoints"

    vars = {
        x = "https://..."
        y = "https://..."
        z = "https://..."
    }

    sensitive_vars = {
        PRIVATE_KEY="some_private_key"
    }
}
```

## Argument Reference
* `name` - (Required) Name of the var(s). This could be anything to uniquely identify the var(s).
* `vars` - Map of vars that are can be outputted in plaintext
* `sensitive_vars` - This is the same as `vars`. The main difference between the two
attributes is `sensitive_vars` outputs are redacted on-screen and replaced by a <sensitive> placeholder, following a terraform
plan or apply. It is recommended to put private keys, passwords, etc in this argument.

## Attributes Reference
The following attributes are exported:
* `id` - The ID of the config

## Import
The `heroku_config` resource is a meta-resource, managed only within Terraform state.
It does not exist as a native Heroku resource. Therefore, it is not possible to import an existing `heroku_config` resource.
