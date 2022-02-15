---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_coupling"
sidebar_current: "docs-heroku-resource-pipeline-coupling"
description: |-
  Provides a Heroku Pipeline Coupling resource.
---

# heroku\_pipeline\_coupling

Provides a [Heroku Pipeline Coupling](https://devcenter.heroku.com/articles/pipelines)
resource.

A pipeline is a group of Heroku apps that share the same codebase. Once a
pipeline is created using [`heroku_pipeline`](./pipeline.html), and apps are added
to different stages using `heroku_pipeline_coupling`, you can promote app slugs
to the downstream stages.

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
}

# Couple apps to different pipeline stages
resource "heroku_pipeline_coupling" "staging" {
  app_id   = heroku_app.staging.id
  pipeline = heroku_pipeline.test-app.id
  stage    = "staging"
}

resource "heroku_pipeline_coupling" "production" {
  app_id   = heroku_app.production.id
  pipeline = heroku_pipeline.test-app.id
  stage    = "production"
}
```

## Argument Reference

The following arguments are supported:

* `app_id` - (Required) Heroku app ID (do not use app name)
* `pipeline` - (Required) The ID of the pipeline to add this app to.
* `stage` - (Required) The stage to couple this app to. Must be one of
`review`, `development`, `staging`, or `production`.

## Attributes Reference

The following attributes are exported:

* `id` - The UUID of this pipeline coupling.

## Import

Pipeline couplings can be imported using the Pipeline coupling `id`, e.g.

```
$ terraform import heroku_pipeline_coupling.foobar 12345678
```