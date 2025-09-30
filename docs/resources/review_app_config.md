---
layout: "heroku"
page_title: "Heroku: heroku_review_app_config"
sidebar_current: "docs-heroku-resource-review-app-config"
description: Provides a resource for configuring review apps.
---

# heroku_review_app_config

Provides a resource for configuring review apps. Using this resource also enables review apps for a pipeline.

-> **IMPORTANT!**
You can only use this resource after you create a pipeline and connect it to a Github repository.
Refer to [the Heroku Dev Center](https://devcenter.heroku.com/articles/github-integration-review-apps#setup)
for more information.

-> **Note:** This resource supports both Cedar and Fir generation pipelines.

## Example Usage

```hcl-terraform
data "heroku_pipeline" "test-pipeline" {
  name = "test pipeline"
}

resource "heroku_review_app_config" "foobar" {
  pipeline_id = data.heroku_pipeline.test-pipeline.id
  org_repo = "yourcompany/yourrepo"
  automatic_review_apps = true
  base_name = "yourcompany"

  deploy_target {
    id   = "us"
    type = "region"
  }

  destroy_stale_apps = true
  stale_days = 5
  wait_for_ci = true
}
```

## Argument Reference

The following arguments are supported:

* `pipeline_id`: (Required) The UUID of an existing pipeline.
* `org_repo`: (Required) The full_name of the repository to enable review apps from.
  Example `heroku/homebrew-brew`.
* `deploy_target`: (Required) The key/value pair to specify whether to use [Common Runtime](https://devcenter.heroku.com/articles/dyno-runtime#common-runtime) or [Heroku Private Spaces](https://devcenter.heroku.com/articles/private-spaces).
  * `id`: (Required) The unique identifier of deploy target.
  * `type`: (Required) The type of deploy target (`space` or `region`).
* `automatic_review_apps`: (Optional) If `true`, triggers the creation of review apps when pull requests
  are opened in the repo. Defaults to `false`.
* `base_name`: (Optional) The unique prefix used to create review app names.
* `destroy_stale_apps: (Optional) If `true`, triggers the automatic deletion of review apps when they’re stale.
  Defaults to `false`.
* `stale_days`: (Optional) The number of days without deploys to destroy stale review apps automatically.
  Must be set with `destroy_stale_apps` and the value must be between `1` and `30` inclusive.
* `wait_for_ci`: (Optional) If true, review apps are only created when CI passes. Defaults to `false`.

## Attributes Reference

The following attributes are exported:

`repo_id`: The ID of the Github repository used for review apps.

## Import

Import an existing review app config with the combination of the pipeline UUID and the Github organization/repository
separated by a colon:

```shell
$ terraform import heroku_review_app_config.foobar afd193fb-7c5a-4d8f-afad-2388f4e6049d:heroku/homebrew-brew
```
