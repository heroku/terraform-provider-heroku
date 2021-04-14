---
layout: "heroku"
page_title: "Heroku: heroku_review_app_config"
sidebar_current: "docs-heroku-resource-review-app-config"
description: |-
Provides a resource for configuring review apps.
---

# heroku_review_app_config

Provides a resource for configuring review apps. Using this resource also enables review apps for a pipeline.

This resource can only be used after the pipeline has been connected to Github repository.
Please visit this [help article](https://devcenter.heroku.com/articles/github-integration-review-apps#setup)
for more information.

## Example Usage.

```hcl-terraform
// Create a new Heroku pipeline
resource "heroku_pipeline" "test-pipeline" {
  name = "test-pipeline"

  owner {
    id = "00b4aef3-073c-425b-92ab-274e483d19db"
    type = "user"
  }
}

// Enable Github integration and connect pipeline to a Github repository.

// Configure review apps
resource "heroku_review_app_config" "foobar" {
  pipeline_id = heroku_pipeline.test-pipeline.id
  org_repo = "yourcompany/yourrepo"
  automatic_review_apps = true
  base_name = "yourcompany"

  deploy_target {
    id = "us"
    type = "region"
  }

  destroy_stale_apps = true
  stale_days = 5
  wait_for_ci = true
}
```

## Argument Reference

The following arguments are supported:

* `pipeline_id` - (Required) The UUID of an existing pipeline.
* `org_repo` - (Required) The full_name of the repository that you want to enable review-apps from.
  Example `heroku/homebrew-brew`.
* `deploy_target` - (Required) Provides a key/value pair to specify whether to use a common runtime or a private space.
  * `id` - (Required) Unique identifier of deploy target.
  * `type` - (Required) Type of deploy target. Must be either `space` or `region`.
* `automatic_review_apps` - (Optional) If true, this will trigger the creation of review apps when pull-requests
  are opened in the repo. Defaults to `false`.
* `base_name` - (Optional) A unique prefix that will be used to create review app names.
* `destroy_stale_apps` - (Optional) If `true`, this will trigger automatic deletion of review apps when they’re stale.
  Defaults to `false`.
* `stale_days` - (Optional) Destroy stale review apps automatically after these many days without any deploys.
  Must be set with `destroy_stale_apps` and value needs to be between `1` and `30` inclusive.
* `wait_for_ci` - (Optional) If true, review apps will only be created when CI passes. Defaults to `false`.

## Attributes Reference

The following attributes are exported:

`repo_id` - ID of the Github repository used for review apps.

## Import

An Existing review app config using the combination of the pipeline UUID and the Github organization/repository
separated by a colon.

```shell
$ terraform import heroku_review_app_config.foobar afd193fb-7c5a-4d8f-afad-2388f4e6049d:heroku/homebrew-brew
```
