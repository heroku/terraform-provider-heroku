---
layout: "heroku"
page_title: "Heroku: heroku_team_members"
sidebar_current: "docs-heroku-datasource-team-members-x"
description: |-
  Get information on members for a Heroku Team.
---

# Data Source: heroku_team_members

Use this data source to get information about members for a Heroku Team.

## Example Usage

```hcl-terraform
data "heroku_team_members" "foobar" {
  team = "name_of_my_heroku_team"
  roles = ["admin", "member", "viewer", "collaborator"]
}
```

## Argument Reference

The following arguments are supported:

* `team` - (Required) The team name.
* `roles` - (Required) List of roles. Acceptable values are `admin`, `member`, `viewer`, `collaborator`, `owner`.
  At least one role must be specified.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the team.
* `members` - All members of the team that have a specified role defined in the `roles` attribute above.
  * `team_member_id` - Unique identifier of the team member on the team.
  * `user_id` - Unique identifier of the team member. This is the member's user ID in Heroku.
  * `email` - Email address of the team member.
  * `role` - Role in the team.
  * `federated` - Whether the user is federated and belongs to an Identity Provider.
  * `two_factor_authentication` - Whether the Enterprise team member has two-factor authentication enabled.