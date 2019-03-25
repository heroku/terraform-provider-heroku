---
layout: "heroku"
page_title: "Heroku: heroku_team_member"
sidebar_current: "docs-heroku-resource-team-member"
description: |-
  Provides the ability to manage members of a Heroku team
---

# heroku\_team\_member

A [Heroku Team Member](https://devcenter.heroku.com/articles/platform-api-reference#team-member) receives access to everything owned by the Team.

To create a Heroku Team, use the [New Team](https://dashboard.heroku.com/teams/new) feature of Heroku Dashboard. For Heroku Enterprise accounts, new Teams may be created within the account by users with the right permissions.

A Heroku "team" was originally called an "organization", and that is still the identifier used elsewhere in this provider. For [`heroku_app`](app.html) & [`heroku_space`](space.html) resources, set the Heroku Team name as the "organization".

## Example Usage

```hcl
# Adds a Heroku user to a Heroku team as a viewer.
resource "heroku_team_member" "foobar-member" {
  team  = "my-team"
  email = "some-user@example.com"
  role  = "member"
}
```

## Argument Reference

* `team` - (Required) The name of the Heroku Team.
* `email` - (Required) Email address of the member
* `role` - (Required) The role to assign the member. See [the API docs](https://devcenter.heroku.com/articles/platform-api-reference#team-member) for available options.

## Import

Team members can be imported using the combination of the team application name, a colon, and the member's email address.

```
$ terraform import heroku_team_member.foobar-member my-team-foobar:some-user@example.com
```
