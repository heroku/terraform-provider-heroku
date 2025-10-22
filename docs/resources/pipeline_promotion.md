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

Use it to perform a pipeline promotion, which deploys a specific release from one app to other apps within the same
pipeline. This operation enables an infrastructure-as-code workflow for promoting code between pipeline stages
such as staging to production. Promotions copy the specified release to all target apps.

->**Note:** Pipeline promotions are immutable. You can't update or modify them after creation.

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

# Access the promoted release information
output "promoted_releases" {
  value = heroku_pipeline_promotion.staging_to_multiple.promoted_release_ids
}

# Access a specific target's release ID
output "production_release_id" {
  value = [
    for release in heroku_pipeline_promotion.staging_to_multiple.promoted_release_ids :
    release.release_id if release.app_id == heroku_app.production.id
  ][0]
}
```

## Argument Reference

The resource supports the following arguments:

* `pipeline`: (Required) The UUID of the pipeline containing the apps.
* `source_app_id`: (Required) The UUID of the source app to promote from.
* `targets`: (Required) The set of UUIDs of target apps to promote to.
* `release_id`: (Required) The UUID of the specific release to promote.


## Attributes Reference

The following attributes are exported:

* `id`: The UUID of the pipeline promotion.
* `status`: The status of the promotion (`pending`, `completed`).
* `created_at`: When the promotion was created.
* `promoted_release_ids`: List of objects containing information about each promoted release. Each object has:
  * `app_id`: The UUID of the target app that received the promotion.
  * `release_id`: The UUID of the release created on that target app.
* `promoted_release_id`: **(Deprecated)** The UUID of the first promoted release. Use `promoted_release_ids` instead. This attribute is maintained for backwards compatibility and will be removed in a future major version.
