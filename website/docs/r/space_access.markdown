---
layout: "heroku"
page_title: "Heroku: heroku_space_access"
sidebar_current: "docs-heroku-resource-space-access"
description: |-
  Provides a Heroku Space access resource for managing access to the space. Note that members with admin privileges will
  only have full access to a Heroku Space.
---

# heroku\_space\_access

Provides a Heroku Space access resource for managing team member permissions to a space. Currently the only supported permission 
is `app_creation`. Note that team admins will already have full access to a Heroku Space.

## Example Usage

```hcl
// Create a new Heroku space
resource "heroku_space" "default" {
  name = "test-space"
  organization = "my-company"
  region = "virginia"
}

// Create a new Heroku app in test-space
resource "heroku_space_access" "member1" {
  space = "${heroku_space.default.name}"
  email = "member1@foobar.com"
  permissions = ["app_creation"]
}

resource "heroku_space_access" "member2" {
  space = "${heroku_space.default.name}"
  email = "member2@foobar.com"
  permissions = ["app_creation"]
}
```

## Argument Reference

The following arguments are supported:

* `space` - (Required) The name of the space.
* `email` - (Required) The email of the team member to set permissions for.
* `permissions` - (Required) The permissions to grant the team member. Currently `app_creation` is the only supported permission.
