---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_"
sidebar_current: "docs-heroku-resource-pipeline-x"
description: |-
  Provides a Heroku Pipeline resource.
---

# heroku\_pipeline


Provides a [Heroku Pipeline](https://devcenter.heroku.com/articles/pipelines)
resource.

A pipeline is a group of Heroku apps that share the same codebase. Once a
pipeline is created, and apps are added to different stages using
[`heroku_pipeline_coupling`](./pipeline_coupling.html), you can promote app
slugs to the next stage.

## Example Usage

```hcl-terraform
# Create Heroku apps for staging and production
resource "heroku_app" "staging" {
  name = "test-app-staging"
  region = "us"
}

resource "heroku_app" "production" {
  name = "test-app-production"
  region = "us"
}

# Create a Heroku pipeline
resource "heroku_pipeline" "test-app" {
  name = "test-app"

  owner {
	id = "16d1c25f-d879-4f4d-ad1b-d807169aaa1c"
	type = "user"
  }
}

# Couple apps to different pipeline stages
resource "heroku_pipeline_coupling" "staging" {
  app      = heroku_app.staging.name
  pipeline = heroku_pipeline.test-app.id
  stage    = "staging"
}

resource "heroku_pipeline_coupling" "production" {
  app      = heroku_app.production.name
  pipeline = heroku_pipeline.test-app.id
  stage    = "production"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the pipeline.

* `owner` - (Required) The owner of the pipeline. This block as the following required attributes:
    * `id` - (Required) The unique identifier (UUID) of a pipeline owner.
    * `type` - (Required) The type of pipeline owner. Can be either `user` or `team`.


Regarding the `owner` attribute block, please note the following:
* The Heroku Platform API allows a pipeline to be created without an owner. However, the UI indicates pipelines require an owner.
So to improve usability, if the `owner` attribute block is not set in your configuration(s), the pipeline owner
will default to the user used to authenticate to the Platform API via this provider.

## Attributes Reference

The following attributes are exported:

* `id` - The UUID of the pipeline.
* `name` - The name of the pipeline.

## Import

Pipelines can be imported using the Pipeline `id`, e.g.

```
$ terraform import heroku_pipeline.foobar 12345678
```
