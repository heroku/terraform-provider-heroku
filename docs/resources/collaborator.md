---
layout: "heroku"
page_title: "Heroku: heroku_collaborator"
sidebar_current: "docs-heroku-resource-collaborator"
description: |-
  Provides the ability to add/remove collaborators from applications that are not owned by a team
---

# heroku\_collaborator

A [Heroku Collaborator](https://devcenter.heroku.com/articles/platform-api-reference#collaborator)
receives access to a specific app that is not owned by a team.

~> **IMPORTANT!:**
This resource only works for apps that are not part of a team.

## Example Usage

```hcl-terraform
resource "heroku_app" "foobar" {
  name = "foobar"
  region = "us"
}

# Create a new collaborator for the foobar application
resource "heroku_collaborator" "foobar-collaborator" {
	app = heroku_app.foobar.id
	email = "collaborator@foobar.com"
}
```

## Argument Reference
* `app` - (Required) Heroku app ID (do not use app name)
* `email` - (Required) Email address of the collaborator

## Attributes Reference
The following attributes are exported:

* `id` - The ID of the collaborator

## Import
Collaborators can be imported using the combination of the application name, a colon, and the collaborator's email address

For example:

```
$ terraform import heroku_collaborator.foobar-collaborator foobar_app:collaborator@foobar.com
```
