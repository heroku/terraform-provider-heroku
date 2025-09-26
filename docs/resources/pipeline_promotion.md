---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_promotion"
sidebar_current: "docs-heroku-resource-pipeline-promotion"
description: |-
  Provides a Heroku Pipeline Promotion resource. Use it to perform deploy a specific release from one app to other apps within the same pipeline.
---

# heroku\_pipeline\_promotion

Provides a [Heroku Pipeline Promotion](https://devcenter.heroku.com/articles/pipelines#promoting)
resource.

Use it to perform a pipeline promotion which deploys a specific release from one app to other apps within the same
pipeline. This operation enables infrastructure-as-code workflow for promoting code between pipeline stages
such as staging to production. Promotions copy the specified release to all target apps.

-> **Notes:** Pipeline promotions are immutable. You can't update or modify them after creation.

## Requirements
* All apps (source and targets) must be in the same pipeline.
* All apps must have the same [generation](https://devcenter.heroku.com/articles/generations) (Cedar or Fir). See [`heroku_pipeline`](./pipeline.html) for generation compatibility requirements.
* The specified release must exist on the source app.

## Example Usage

```hcl
# Basic promotion from staging to production
resource "heroku_pipeline_promotion" "staging_to_prod" {
  pipeline      = heroku_pipeline.my_app.id
  source_app_id = heroku_app.staging.id
  release_id    = "01234567-89ab-cdef-0123-456789abcdef"
  targets       = [heroku_app.production.id]
}

# Promotion to multiple target apps
resource "heroku_pipeline_promotion" "staging_to_multiple" {
  pipeline      = heroku_pipeline.my_app.id
  source_app_id = heroku_app.staging.id
  release_id    = "01234567-89ab-cdef-0123-456789abcdef"
  targets       = [ 
    heroku_app.production.id,
    heroku_app.demo.id
  ]
}
```

## Argument Reference

The following arguments are supported:

* `pipeline`: (Required) The UUID of the pipeline containing the apps.
* `source_app_id`: (Required) The UUID of the source app to promote from.
* `targets`: (Required) The set of UUIDs of target apps to promote to.
* `release_id`: (Required) The UUID of the specific release to promote.

## Attributes Reference

The following attributes are exported:

* `id`: The UUID of the pipeline promotion.
* `status`: The status of the promotion (`pending`, `completed`).
* `created_at`: When the promotion was created.
* `promoted_release_id`: The UUID of the release that was actually promoted.
