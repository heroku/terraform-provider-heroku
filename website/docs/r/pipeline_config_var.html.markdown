---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_config_var"
sidebar_current: "docs-heroku-resource-pipeline-config-var"
description: |-
  Provides a Heroku Pipeline Config Var resource.
---

# heroku\_pipeline\_config\_var

Provides a resource to manage a pipeline's config vars.

~> **NOTE:** The pipeline config var API only relates to the config vars that can be set at the pipeline level.
These two supported stages are:
- Heroku CI config vars (test stage)
- Review Apps config vars (review stage)

The development, staging & production stages do not have stage-level config vars, only those on the apps within each stage.

## Example Usage

```hcl
# Create a Heroku pipeline
resource "heroku_pipeline" "test-pipeline" {
  name = "test-app"
}

resource "heroku_pipeline_config_var" "configs" {
  pipeline_id = heroku_pipeline.test-pipeline.id
  pipeline_stage = "test"
  
  vars = {
    ENV = "test"
    TARGET = "develop"
  }

  sensitive_vars = {
    TEST_ACCESS_TOKEN = "some_access token"
  }
}
```

## Argument Reference

The following arguments are supported:

* `pipeline_id` - (Required) The UUID of an existing pipeline.

* `pipeline_stage` - (Required) The pipeline's stage. Supported values are `test` & `review`.

* `vars` - Map of config vars that can be output in plaintext.

* `sensitive_vars` - This is the same as `vars`. The main difference between the two attributes is `sensitive_vars` outputs
are redacted on-screen and replaced by a `<sensitive>` placeholder, following a terraform `plan` or `apply`.
It is recommended to put private keys, passwords, etc in this argument.

## Attributes Reference

The following attributes are exported:

* `all_vars` - All vars of a pipeline stage. This is marked `sensitive` so that `sensitive_vars` do not leak in the console/logs.

## Import
This resource defines two config var attributes with one of them used for masking any sensitive/secret variables
during a `terraform plan|apply` in a CI build, terminal, etc. This 'sensitive' distinction for config vars is unique to
this provider and not a built-in feature of the Heroku Platform API. Therefore, it will not be possible to import
this resource.

However, it is safe to define the resource in your configuration file and execute a `terraform apply`
as the end result is `noop` when the config vars already exist on the remote resource.
