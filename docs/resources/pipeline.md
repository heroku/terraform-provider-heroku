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

A pipeline is a group of Heroku apps that share the same codebase. After creating a
pipeline, and adding apps to different stages using
[`heroku_pipeline_coupling`](./pipeline_coupling.html), you can promote app
build artifacts to the next stage.

## Generation Compatibility

All apps in a pipeline must use the same Heroku platform [generation](https://devcenter.heroku.com/articles/generations) (Cedar or Fir). 
Attempting to add apps from different generations results in an error.

## Ownership and Access

You can create pipelines for personal or team resources. Access to a pipeline
is based on access to the apps in the pipeline.

For team pipelines, configure auto-join settings in the Heroku Dashboard's
**`Pipeline Access`** section.

## GitHub Connection

You can only connect pipelines to GitHub via Heroku CLI or the dashboard web UI.

If your Terraform use case requires a GitHub connection, create the pipeline 
manually, copy its ID (UUID) from its dashboard URL, and then reference that ID in 
the Terraform configuration.

## Empty Pipelines

You can create empty pipelines via the Heroku Dashboard. Only the pipeline creator
can access an empty pipeline in Heroku CLI and dashboard.

You must identify empty pipelines in API queries via ID (UUID).

Team members can access empty team pipelines via API. This access allows
manually created pipelines to be populated with app couplings via Terraform.

Removing all app couplings from a pipeline automatically deletes 
the empty pipeline, within a short period of time (less than one hour).

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

The resource supports the following arguments:

* `name`: (Required) The name of the pipeline.
* `owner`: (Required) The owner of the pipeline. This block has the following required attributes:
    * `id`: (Required) The unique identifier (UUID) of a pipeline owner.
    * `type`: (Required) The type of pipeline owner ( `user` or `team`).


For the `owner` attribute block:

* You can create unowned pipelines with the Heroku Platform API. However, the dashboard UI requires that pipelines have an owner.
* To improve usability, if you don't set the `owner` attribute block in your configuration(s), the pipeline owner
defaults to the user used to authenticate to the Platform API via this provider.

## Attributes Reference

The following attributes are exported:

* `id`: The UUID of the pipeline.
* `name`: The name of the pipeline.

## Import

Import pipelines using the pipeline's `id`.

For example:

```
$ terraform import heroku_pipeline.foobar 12345678
```
