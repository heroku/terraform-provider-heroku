---
layout: "heroku"
page_title: "Heroku: heroku_team"
sidebar_current: "docs-heroku-resource-team"
description: |-
  Provides the ability to create a Heroku team
---

# heroku\_team
Provides a [Heroku Team](https://devcenter.heroku.com/articles/platform-api-reference#team) resource.

~> **NOTE:** This resource can only create a [Heroku Team](https://devcenter.heroku.com/articles/heroku-teams),
not an [Enterprise Team](https://devcenter.heroku.com/categories/enterprise-teams). However, a user can manage an
existing enterprise team via `terraform import`.

## Example Usage
```hcl
resource "heroku_team" "foobar" {
  name  = "my-team"
}
```

## Argument Reference
* `name` - (Required) Unique name of team.

## Attributes Reference
The following attributes are exported:
* `id` - The UUID of the team
* `default` - Whether to use this team when none is specified
* `credit_card_collections` - Whether charges incurred by the team are paid by credit card
* `membership_limit` - Upper limit of members allowed in a team
* `provisioned_licenses` - Whether the team is provisioned licenses by Salesforce

## Import
Both regular and enterprise teams can be imported with the following sample command:

```
$ terraform import heroku_team.foobar name-of-my-existing-team
```
