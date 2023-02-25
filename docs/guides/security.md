---
layout: "heroku"
page_title: "Heroku: Secure Practices"
sidebar_current: "docs-heroku-guides-security"
description: |-
  Guide to using the provider securely.
---

# Authentication

The API key used by Terraform must inherently have complete permission 
to manage Heroku resources.

To generate API keys with minimal scope, see 
[Dev Center article **Using Terraform with Heroku: Authorization**](https://devcenter.heroku.com/articles/using-terraform-with-heroku#authorization).

The API key can be set for the provider following the 
[Provider Authentication docs](../#authentication).

# Sensitivity

Terraform includes the concept of `sensitive` values which are 
automatically redacted from terminal output, such as plan diffs and 
output summaries.

Various resource attributes are defined in the provider as sensitive, 
including: `heroku_app`#`all_config_vars`, 
`heroku_addon`#`config_var_values`, & `heroku_app_webhook`#`secret`.

In every configuration, practice marking `sensitive = true` variables &
outputs that contain secret data:

```hcl
variable "heroku_api_key" {
  type        = string
  sensitive   = true
}

output "production_database_url" {
  type      = string
  value     = heroku_addon.production_postgres.config_var_values["DATABASE_URL"]
  sensitive = true
}
```

# Config Vars

Especially sensitive Heroku app config vars may be managed from outside of 
Terraform, set through `heroku config` CLI, web dashboard, or Platform API,
to avoid their values touching Terraform workflows.

Also, config vars automatically set by add-ons, such as Postgres 
`DATABASE_URL`, will be recorded in Terraform state as part of the standard
functionality of this Terraform provider.

In high-security situations, these externally managed config vars can be 
completely excluded from Terraform by setting the 
[provider attributes](../#argument-reference):

```hcl
provider "heroku" {
  customizations {
    set_app_all_config_vars_in_state = false
    set_addon_config_vars_in_state   = false
  }
}
```

As a result, `heroku_app`#`all_config_vars` and 
`heroku_addon`#`config_var_values` will be empty for all resources 
managed in Terraform.

# Logging

In normal runtime, the provider is designed to avoid logging sensitive data.

When `TF_LOG` environment variable is set, such as `TF_LOG=debug`, the 
provider will log extensive data including Heroku API calls. `Authorization` 
headers are automatically redacted, but logged request and response JSON
bodies will contain secret values, such as app config vars.

Only set `TF_LOG` in environments where the sensitive log output is 
acceptable. Destroy/delete such logs after use to avoid disclosure.
