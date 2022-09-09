---
layout: "heroku"
page_title: "Heroku: heroku_pipeline"
sidebar_current: "docs-heroku-datasource-pipeline-x"
description: |-
  Get information on a Heroku Pipeline.
---

# Data Source: heroku_pipeline

Use this data source to get information about a Heroku Pipeline.

~> **NOTE:**
This data source can only be used to fetch information regarding a pipeline that has apps already associated to it.
This is a limitation in the Heroku Platform API where it is not possible to query a pipeline without apps by its name.

## Example Usage

```hcl-terraform
data "heroku_pipeline" "foobar" {
  name = "pipeline-test-123"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The pipeline name or ID. Empty pipelines can only be retrieved by ID (UUID).

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the pipeline
* `owner_id` - The pipeline owner's ID
* `owner_type` - The pipeline owner's type
