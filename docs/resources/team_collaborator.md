---
layout: "heroku"
page_title: "Heroku: heroku_team_collaborator"
sidebar_current: "docs-heroku-resource-team-collaborator"
description: |-
  Provides the ability to add/edit/remove team collaborators from team applications
---

# heroku\_team\_collaborator

A [Heroku Team Collaborator](https://devcenter.heroku.com/articles/platform-api-reference#team-app-collaborator)
receives access to a specific Team-owned app.

To create a Heroku Team, use the [New Team](https://dashboard.heroku.com/teams/new) feature of Heroku Dashboard.
For Heroku Enterprise accounts, new Teams may be created within the account by users with the right permissions.

A Heroku "team" was originally called an "organization", and that is still the identifier used elsewhere in this provider.
For [`heroku_app`](app.html) & [`heroku_space`](space.html) resources, set the Heroku Team name as the "organization".

~> **IMPORTANT:**
This resource only works for Team-owned apps.

## Example Usage

```hcl-terraform
resource "heroku_app" "foobar" {
  name = "some name"
  region = "us"
  organization {
    name = "some organization"
  }
}

# Create a new team collaborator for the foobar application that has view, operate, manage permissions
resource "heroku_team_collaborator" "foobar-collaborator" {
	app = heroku_app.foobar.name
	email = "collaborator@foobar.com"
	permissions = ["view", "operate", "manage"]
}
```

## Argument Reference
* `app` - (Required) The name of the team app that the team collaborator will be added to.
* `email` - (Required) Email address of the team collaborator
* `permissions` - (Required) List of permissions that will be granted to the team collaborator. The order in which
individual permissions are set here does not matter. Please [visit this link](https://devcenter.heroku.com/articles/app-permissions)
for more information on available permissions.

## Attributes Reference
The following attributes are exported:

* `id` - The ID of the team collaborator

## Import
Team Collaborators can be imported using the combination of the team application name, a colon, and the collaborator's email address

For example:

```
$ terraform import heroku_team_collaborator.foobar-collaborator foobar_app:collaborator@foobar.com
```
