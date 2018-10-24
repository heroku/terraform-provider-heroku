---
layout: "heroku"
page_title: "Heroku: heroku_account_feature"
sidebar_current: "docs-heroku-resource-addon"
description: |-
  Provides a Heroku Account Feature resource.

# heroku\_account_feature

Provides a Heroku Account Feature resource. An account feature represents a Heroku labs capability
that can be enabled or disabled for an account on Heroku.

~> **NOTE:** If this resource's HCL is removed from a `.tf` file, the behavior is to disable account feature
and remove resource from state.

## Example Usage

```hcl
# Create a new Heroku app
resource "heroku_account_feature" "foobar" {
  name = "foobar"
  enabled = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the account feature
* `enabled` - (Required) whether or not account feature has been enabled

## Attributes Reference

The following attributes are exported:

* `id` - Comprised of acount email & feature name
* `description` - description of account feature
* `state` - state of account feature

## Import

Existing account feature can be imported using a combination of the account email (the email address tied to the Heroku API key)
and the feature name

For example:
```
$ terraform import heroku_account_formation.foobar email@email.com:foobar-feature
```
