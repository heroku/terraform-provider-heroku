---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_promotion_"
sidebar_current: "docs-heroku-resource-pipeline-promotion-x"
description: |-
  Provides a Heroku Pipeline Promotion resource.
---

# heroku\_pipeline\_promotion

Provides a [Heroku Pipeline Promotion](https://devcenter.heroku.com/articles/pipelines#deployment-with-pipelines)
resource.

A pipeline is a group of Heroku apps that share the same codebase. Once a
pipeline is created, and apps are added to different stages using
[`heroku_pipeline_coupling`](./pipeline_coupling.html), you can promote app
slugs to the next stage or within a given stage.

Note the lack of an Update or Destroy operation within this resource. This 
Resource will primarily be used to perform an initial deploy of an app when
standing up infrastructure with Terraform. Generally speaking, it will not 
be used to manage app versions over time as they change.

## Example Usage

```hcl
# Create a source app (the app we will promote from)
resource "heroku_app" "foobar-source-app" {
	name = "test-source-app"
	region = "us"
}

# Create target apps (the apps we will promote to)
resource "heroku_app" "foobar-target-app1" {
	name = "test-target-app1"
	region = "us"
}
resource "heroku_app" "foobar-target-app2" {
	name = "test-target-app1"
	region = "us"
}

# Create a Heroku pipeline
resource "heroku_pipeline" "foobar-pipeline" {
	name = "test-pipeline"

	owner {
		id = "16d1c25f-d879-4f4d-ad1b-d807169aaa1c"
		type = "user"
	}
}

# Create an app release for the source app
resource "heroku_app_release" "foobar-release" {
	app = "${heroku_app.foobar-source-app.id}"
	slug_id = "01234567-89ab-cdef-0123-456789abcdef"
}

# Create pipeline couplings for each app in the pipeline
resource "heroku_pipeline_coupling" "foobar-source-coupling" {
	app      = "${heroku_app.foobar-source-app.id}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "staging"
}
resource "heroku_pipeline_coupling" "foobar-target-coupling1" {
	app      = "${heroku_app.foobar-target-app1.id}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "production"
}
resource "heroku_pipeline_coupling" "foobar-target-coupling2" {
	app      = "${heroku_app.foobar-target-app2.id}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "production"
}

# Create a pipeline promotion that will promote the source app to the target apps.
resource "heroku_pipeline_promotion" "foobar-promotion" {
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	source = "${heroku_app.foobar-source-app.uuid}"
	targets = ["${heroku_app.foobar-target-app1.uuid}","${heroku_app.foobar-target-app2.uuid}"]
}
```

## Argument Reference

The following arguments are supported:

* `pipeline` - (Required) The ID of the pipeline.

* `source` - (Required) The unique identifier (UUID) of the source app.

* `targets` - (Required) The unique identifier (UUID) of each of the target apps.

## Attributes Reference

The following attributes are exported:

* `pipeline` - The UUID of the pipeline promotion.
* `source` - The unique identifier (UUID) of the source app.
* `targets` - The unique identifiers (UUID) of each of the target apps.
* `release_id` - The app release ID of the source and target apps after a successful promotion.
* `status` - The status of the pipeline promotion operation.
* `created_at` - The datetime stamp of when the pipeline promotion was created.
* `updated_at` - The datetime stamp of when the pipeline promotion was updated.

## Import

Pipeline promotions cannot be imported.
