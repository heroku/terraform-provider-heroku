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

## Generation Compatibility

All apps in a pipeline must use the same Heroku platform generation (Cedar or Fir). 
Attempting to add apps from different generations will result in an error.

## Ownership & Access

Pipelines may be created as Personal or Team resources. Access to a pipeline
is based on access to apps in the pipeline.

For team pipelines, auto-join settings are available in the Heroku Dashboard's
Pipeline Access section.

## GitHub Connection

Pipelines may only be connected to GitHub via Heroku CLI or Dashboard web UI.

If your Terraform use-case requires GitHub connection, then create the pipeline 
manually, copy its ID (UUID) from its Dashboard URL, and then reference that ID in 
the Terraform configuration.

## Empty Pipelines

Pipelines created via Heroku Dashboard may be empty. Only the pipeline creator
can access an empty pipeline in Heroku CLI and Dashboard.

Empty pipelines must be identified in API queries via ID (UUID).

Empty team pipelines may be accessed by team members via API. This permits
manually created pipelines to be populated with app couplings via Terraform.

Removing all app couplings from a pipeline will result in automatic deletion of 
the empty pipeline, within a short period of time (less than one-hour).

## Example Usage

```hcl-terraform
# Create Heroku apps for staging and production
resource "heroku_app" "staging" {
  name   = "test-app-staging"
  region = "us"
}

resource "heroku_app" "production" {
  name   = "test-app-production"
  region = "us"
}

# Create a Heroku pipeline
resource "heroku_pipeline" "test" {
  name = "test-pipeline"

  owner {
	  id   = "00b4aef3-073c-425b-92ab-274e483d19db"
	  type = "team"
  }
}

# Couple apps to different pipeline stages
resource "heroku_pipeline_coupling" "staging" {
  app_id   = heroku_app.staging.id
  pipeline = heroku_pipeline.test.id
  stage    = "staging"
}

resource "heroku_pipeline_coupling" "production" {
  app_id   = heroku_app.production.id
  pipeline = heroku_pipeline.test.id
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
