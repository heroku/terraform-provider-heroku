---
layout: "heroku"
page_title: "Heroku: heroku_space_app_access"
sidebar_current: "docs-heroku-resource-space-app-access"
description: |-
  Provides a Heroku Space App Access resource for managing permissions within the Private Space.
---

# heroku\_space\_app\_access

Provides a resource for managing permissions for the entire Private Space. Members with the admin role will
always have full permissions in the Private Space, so using this resource on an admin will have no effect.
The provided email must already be a member of the Heroku Team. Currently, the only supported permission is `create_apps`.

## Example Usage

```hcl-terraform
// Create a new Heroku Private Space
resource "heroku_space" "default" {
  name = "test-space"
  organization = "my-company"
  region = "virginia"
}

// Give an existing team member create_apps permissions to the Private Space
resource "heroku_space_app_access" "member1" {
  space = heroku_space.default.name
  email = "member1@example.com"
  permissions = ["create_apps"]
}

// Remove all permissions from an existing team member
resource "heroku_space_app_access" "member2" {
  space = heroku_space.default.name
  email = "member2@example.com"
  permissions = []
}
```

## Argument Reference

The following arguments are supported:

* `space` - (Required) The name of the Private Space (ID/UUID is acceptable too, but must be used consistently).
* `email` - (Required) The email of the existing Heroku Team member.
* `permissions` - (Required) The permissions to grant the team member for the Private Space.
  Currently `create_apps` is the only supported permission. If not provided the member will have no permissions to the space.
  Members with admin role will always have `create_apps` permissions, which cannot be removed.

## Importing

Existing permissions can be imported using the combination of the Private Space name, a colon, and the member email.

For example:

```
$ terraform import heroku_space_app_access.member1 my-space:member1@foobar.com
```