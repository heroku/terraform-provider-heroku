---
layout: "heroku"
page_title: "Heroku: heroku_space_member"
sidebar_current: "docs-heroku-resource-space-member"
description: |-
  Provides a Heroku Space member resource for managing permissions to the space. Note that members with the admin role will
  always have full permissions to a Heroku Space. The provided member must already exist in your Heroku organization.
---

# heroku\_space\_member

Provides a Heroku Space member resource for managing permissions to the space. Note that members with the admin role will always have full permissions to a Heroku Space. The provided member must already exist in your Heroku organization. Currently the only supported permission is `create_apps`.

## Example Usage

```hcl
// Create a new Heroku space
resource "heroku_space" "default" {
  name = "test-space"
  organization = "my-company"
  region = "virginia"
}

// Give an existing team member create_apps permissions to the space
resource "heroku_space_member" "member1" {
  space = "${heroku_space.default.name}"
  email = "member1@foobar.com"
  permissions = ["create_apps"]
}

// Remove an existing team members permissions to the space
resource "heroku_space_member" "member2" {
  space = "${heroku_space.default.name}"
  email = "member2@foobar.com"
}
```

## Argument Reference

The following arguments are supported:

* `space` - (Required) The name of the space.
* `email` - (Required) The email of the team member to set permissions for.
* `permissions` - (Optional) The permissions to grant the team member. Currently `create_apps` is the only supported permission. If not provided the member will have no permissions to the space (Note that members with the admin role are unaffected)
