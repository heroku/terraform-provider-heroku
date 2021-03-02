---
layout: "heroku"
page_title: "Heroku: heroku_account_feature"
sidebar_current: "docs-heroku-resource-account-feature"
description: |-
  Provides a resource to create and manage User Features on Heroku.
---

# heroku\_account\_feature

This resource is used to create and manage [User Features](https://devcenter.heroku.com/articles/heroku-beta-features) on Heroku.

~> **NOTE:** If this resource's HCL is removed from a `.tf` file, the behavior is to disable account feature
and remove resource from state.

## Available Features

For a list of available features, use the [`heroku labs`](https://devcenter.heroku.com/articles/heroku-cli)
command to fetch them for the current authenticated user.

The output will contain **User Features** that may be managed with this resource.

## Example Usage

```hcl-terraform
resource "heroku_account_feature" "example_metrics" {
  name = "metrics-request-volume"
  enabled = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the account feature
* `enabled` - (Required) Enable or disable the account feature

## Attributes Reference

The following attributes are exported:

* `id` - Comprised of acount email & feature name
* `description` - Description of account feature
* `state` - State of account feature

## Import

Existing account features can be imported using a combination of the account email (the email address tied to the Heroku API key)
and the feature name.

For example:

```
$ terraform import heroku_account_feature.example_metrics name@example.com:metrics-request-volume
```
