---
layout: "heroku"
page_title: "Heroku: heroku_app_config_association"
sidebar_current: "docs-heroku-resource-app-config-association"
description: |-
  Provides a Heroku App Config Association resource, making it possible set, update, and remove Heroku app config vars 
---

# heroku\_app\_config\_association
Provides a Heroku App Config Association resource, making it possible to set/update/remove heroku app config vars independently from
`heroku_app`. An example usage scenario could be:

- User has separate git repositories for various micro-services. Multiple micro-services use Kafka.
- User has a separate repository for kafka terraform files with blue/green support.
- User builds out new clusters.
- Prior to this resource's introduction, user would need one `terraform apply` to update state and X number of `terraform apply`
for each micro-service to pick up the new kafka clusters. However with this resource, user can do one `terraform apply`
and let Heroku handle the rolling restarts to pick up the new config vars.

~> **NOTE:**
- Heroku does not have a 'sensitivity' distinction for its config variables.
This distinction is only made during Terraform plans and applies to avoid leaking sensitive data via plan output.
- Be careful when having config variables defined in both `heroku_app` and `heroku_app_config_association` resources. As the latter resource
has a dependency on the former, any overlapping config variables in `heroku_app` will be overwritten in `heroku_app_config_association`.

## Example HCL
```hcl
resource "heroku_config" "common" {
    name = "common-vars"

    vars = {
        LOG_LEVEL = "info"
    }

    sensitive_vars = {
        PRIVATE_KEY="some_private_key"
    }
}

resource "heroku_app" "foobar" {
  name   = "my-cool-app"
  region = "us"
}

resource "heroku_app" "foobar2" {
  name   = "my-cool-app2"
  region = "us"
}

resource "heroku_app_config_association" "foobar" {
  app_id = "${heroku_app.foobar.id}"

  vars = "${heroku_config.common.vars}"
  sensitive_vars = "${heroku_config.common.sensitive_vars}"
}

resource "heroku_app_config_association" "foobar2" {
  app_id = "${heroku_app.foobar2.id}"

  vars = "${heroku_config.common.vars}"
  sensitive_vars = {
    DATABASE_URL = "some_db_url_that_has_auth_info"
  }
}
```

## Argument Reference
* `app_id` - (Required) A Heroku app's `UUID`. Can also be the name of the Heroku app but `UUID` is preferred as it is idempotent
* `vars` - Map of config vars that are can be outputted in plaintext
* `sensitive_vars` - This is the same as `vars`. The main difference between the two
attributes is `sensitive_vars` outputs are redacted on-screen and replaced by a <sensitive> placeholder, following a terraform
plan or apply. It is recommended to put private keys, passwords, etc in this argument.

## Attributes Reference
The following attributes are exported:
* `id` - The ID of the app config association

## Import
The `heroku_app_config_association` resource's primary attributes are managed only within Terraform state.
It does not exist as a native Heroku resource. Therefore, it is not possible to import an existing `heroku_app_config_association` resource.
