---
layout: "heroku"
page_title: "Heroku: heroku_app_config_association"
sidebar_current: "docs-heroku-resource-app-config-association"
description: |-
  Provides a Heroku App Config Association resource, making it possible set, update, and remove Heroku app config vars 
---

# heroku\_app\_config\_association
Provides a Heroku App Config Association resource, making it possible to set/update/remove heroku app config vars independently from
the `heroku_app` resource. An example usage scenario could be:

* User has separate git repositories for various micro-services. Multiple micro-services use Kafka.
* User has a separate repository for kafka terraform files with blue/green support.
* User builds out new clusters.
* Prior to this resource's introduction, user would need one `terraform apply` to update state and X number of `terraform apply`
for each micro-service to pick up the new kafka clusters. However with this resource, user can do one `terraform apply`
and let Heroku handle the rolling restarts to pick up the new config vars.

### "Sensitive" is not secret
Heroku does not have a 'sensitivity' distinction for its config variables.
This distinction is only made during terraform `plan` and `apply` to avoid leaking sensitive data in the console output.

### Beware of conflicting vars
Be careful when having config variables defined in both `heroku_app` and `heroku_app_config_association` resources. As the latter resource
has a dependency on the former, any overlapping config variables in `heroku_app` will be overwritten in `heroku_app_config_association`
during a `terraform apply`. Furthermore, this overlap will cause an infinite dirty terraform plan if config variables have
different values on both resources at the same time. It is recommended to use one or the other resource, not both, to manage your app(s) config vars.

## Example HCL
```hcl-terraform
resource "heroku_config" "common" {
    vars = {
        LOG_LEVEL = "info"
    }

    sensitive_vars = {
        PRIVATE_KEY = "some_private_key"
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
  app_id = heroku_app.foobar.id

  vars = heroku_config.common.vars
  sensitive_vars = heroku_config.common.sensitive_vars
}

resource "heroku_app_config_association" "foobar2" {
  app_id = heroku_app.foobar2.id

  vars = heroku_config.common.vars
  sensitive_vars = {
    DATABASE_URL = "some_db_url_that_has_auth_info"
  }
}
```

## Argument Reference

* `app_id` - (Required) A Heroku app's `UUID`. Can also be the name of the Heroku app but `UUID` is preferred as it is idempotent.
* `vars` - Map of config vars that can be output in plaintext.
* `sensitive_vars` - This is the same as `vars`. The main difference between the two
attributes is `sensitive_vars` outputs are redacted on-screen and replaced by a <sensitive> placeholder, following a terraform
plan or apply. It is recommended to put private keys, passwords, etc in this argument.

## Attributes Reference
The following attributes are exported:

* `id` - The ID of the app config association.

## Import
This resource defines two config var attributes with one of them used for masking any sensitive/secret variables
during a `terraform plan|apply` in a CI build, terminal, etc. This 'sensitive' distinction for config vars is unique to
this provider and not a built-in feature of the Heroku Platform API. Therefore, it will not be possible to import
this resource.

However, it is safe to define the resource in your configuration file and execute a `terraform apply`
as the end result is `noop` when the config vars already exist on the remote resource.
