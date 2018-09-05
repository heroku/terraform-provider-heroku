---
layout: "heroku"
page_title: "Heroku: heroku_team_member"
sidebar_current: "docs-heroku-resource-team-member"
description: |-
  Provides the ability to manage members of a Heroku team
---

# heroku\_team\_member

Provides a [Heroku Team Collaborator](https://devcenter.heroku.com/articles/platform-api-reference#team-member) resource.

~> **NOTE:** Please only use this resource if you have team/organization apps

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

* `team` - (Required) The name of the Heroku team that the team member will be added to.
* `email` - (Required) Email address of the team collaborator
* `role` - (Required) The role to assign the team member. See [the API docs](https://devcenter.heroku.com/articles/platform-api-reference#team-member) for available options.

## Import

Team members can be imported using the combination of the team application name, a colon, and the member's email address.

```
$ terraform import heroku_team_member.foobar-member my-team-foobar:some-user@example.com
```
