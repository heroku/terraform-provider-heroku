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

A pipeline is a group of Heroku apps that share the same codebase. Use a pipeline coupling to add apps to different stages of the pipeline.

After creating a
pipeline with [`heroku_pipeline`](./pipeline.html), and adding apps
to stages with `heroku_pipeline_coupling`, you can [promote](/pipeline_promotion.html) an app's build artifacts
to the downstream stages.

See [`heroku_pipeline`](./pipeline.html) for complete usage documentation.

## Example Usage

```hcl-terraform
resource "heroku_pipeline_coupling" "production" {
  app_id   = heroku_app.production.id
  pipeline = heroku_pipeline.test.id
  stage    = "production"
}
```

## Argument Reference

The following arguments are supported:

* `app_id`: (Required) The Heroku app ID (not name)
* `pipeline`: (Required) The ID of the pipeline to add this app to.
* `stage`: (Required) The stage to couple this app to (`review`, `development`, `staging`, or `production`).

## Attributes Reference

The following attributes are exported:

* `id`: The UUID of the pipeline coupling.

## Import

You can import a pipeline couplings with the its `id`, e.g.

```
$ terraform import heroku_pipeline_coupling.foobar 12345678
```
