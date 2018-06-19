---
layout: "heroku"
page_title: "Heroku: heroku_team_collaborator"
sidebar_current: "docs-heroku-resource-team-collaborator"
description: |-
  Provides the ability to add/edit/remove team collaborators from team applications
---

# heroku\_team\_collaborator

Provides a [Heroku Team Collaborator](https://devcenter.heroku.com/articles/platform-api-reference#team-app-collaborator)
resource.

A team collaborator represents an account that has been given access to a team app on Heroku.

~> **NOTE:** Please only use this resource if you have team/organization apps

## Example Usage

```hcl
# Create a new team collaborator for the foobar application that has view, operate, manage permissions
resource "heroku_team_collaborator" "foobar-collaborator" {
	app = "${heroku_app.foobar.name}"
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
