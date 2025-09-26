---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_promotion"
sidebar_current: "docs-heroku-resource-pipeline-promotion"
description: |-
  Provides a Heroku Pipeline Promotion resource.
---

# heroku\_pipeline\_promotion

Provides a [Heroku Pipeline Promotion](https://devcenter.heroku.com/articles/pipelines)
resource.

A pipeline promotion allows you to deploy releases from one app to other apps within the same
pipeline. This enables infrastructure-as-code workflow for promoting code between pipeline stages
such as staging to production.

You can promote either the latest release from the source app, or specify a particular release by ID.

## Example Usage

```hcl
# Basic promotion from staging to production (latest release)
resource "heroku_pipeline_promotion" "staging_to_prod" {
  pipeline      = heroku_pipeline.my_app.id
  source_app_id = heroku_app.staging.id
  targets       = [heroku_app.production.id]
}

# Promotion of a specific release
resource "heroku_pipeline_promotion" "specific_release" {
  pipeline      = heroku_pipeline.my_app.id
  source_app_id = heroku_app.staging.id
  release_id    = "01234567-89ab-cdef-0123-456789abcdef"
  targets       = [heroku_app.production.id]
}

# Promotion to multiple target apps
resource "heroku_pipeline_promotion" "staging_to_multiple" {
  pipeline      = heroku_pipeline.my_app.id
  source_app_id = heroku_app.staging.id
  targets       = [
    heroku_app.production.id,
    heroku_app.demo.id
  ]
}
```

## Argument Reference

The following arguments are supported:

* `pipeline` - (Required) The UUID of the pipeline containing the apps.
* `source_app_id` - (Required) The UUID of the source app to promote from.
* `targets` - (Required) Set of UUIDs of target apps to promote to.
* `release_id` - (Optional) The UUID of a specific release to promote. If not specified, promotes the latest release from the source app.

## Attributes Reference

The following attributes are exported:

* `id` - The UUID of this pipeline promotion.
* `status` - The status of the promotion (`pending`, `completed`).
* `created_at` - When the promotion was created.
* `promoted_release_id` - The UUID of the release that was actually promoted.

## Notes

* Pipeline promotions are immutable - they cannot be updated or modified after creation.
* All apps (source and targets) must be in the same pipeline.
* All apps must have the same generation (Cedar or Fir). See [`heroku_pipeline`](./pipeline.html) for generation compatibility requirements.
* The source app must have at least one release to promote.
* Promotions copy either the latest release (if no `release_id` specified) or the specified release to all target apps.

