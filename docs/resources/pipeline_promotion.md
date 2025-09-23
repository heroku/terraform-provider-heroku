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

Currently promotes the latest release from the source app. Support for promoting specific releases
(`release_id` parameter) requires additional API support from the Heroku platform team.

## Example Usage

```hcl
# Basic promotion from staging to production
resource "heroku_pipeline_promotion" "staging_to_prod" {
  pipeline      = heroku_pipeline.my_app.id
  source_app_id = heroku_app.staging.id
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
* `release_id` - (Optional) **Not yet supported**. The UUID of a specific release to promote. 
  Currently returns an error as this requires additional Heroku platform API support.

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
* Promotions copy the latest release from the source app to all target apps.

## Future Enhancement

The `release_id` parameter will be supported once the Heroku platform team adds the necessary API
functionality. This will enable promoting specific releases rather than just the latest release.
