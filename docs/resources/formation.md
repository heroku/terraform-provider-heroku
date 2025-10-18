---
layout: "heroku"
page_title: "Heroku: heroku_formation"
sidebar_current: "docs-heroku-resource-formation"
description: |-
  Provides the ability to update the formation of a heroku app that has a running dyno.
---

# heroku\_formation

Provides a [Heroku Formation](https://devcenter.heroku.com/articles/platform-api-reference#formation)
resource.

A formation represents the formation of processes that should be set for an application.

Please note the following:
* The application must have a dyno in order to update its formation.
* If the heroku formation resource is removed and deleted, this will be a no-op action in Heroku.
The Heroku Platform does not have a `DELETE` endpoint for `formation`.
* This resource works well with the `heroku_app_release` resource, which allows you to deploy a slug/release to an application
before the formation can be updated.

## Example Usage

```hcl-terraform
# Creates a new application called foobar
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

# Creates a new release for application foobar using a slug id
resource "heroku_app_release" "foobar-release" {
    app_id = heroku_app.foobar.id
    slug_id = "01234567-89ab-cdef-0123-456789abcdef"
}

# Update the web formation for the foobar application's web
resource "heroku_formation" "foobar-web" {
    app_id = heroku_app.foobar.id
    type = "web"
    quantity = 2
    size = "standard-2x"

    # Tells Terraform that this formation must be created/updated only after the app release has been created
    depends_on = ["heroku_app_release.foobar-release"]
}
```

## Argument Reference

* `app_id` - (Required) Heroku app ID (do not use app name)
* `type` - (Required) type of process such as "web"
* `quantity` - (Required) number of processes to maintain
* `size` - (Required) dyno size. Capitalization does not matter. Examples:
  * Cedar apps: `"standard-1x"`, `"standard-2x"`, `"performance-m"`, `"private-s"`, etc.
  * Fir apps: `"dyno-2c-1gb"`, `"dyno-4c-2gb"`, etc.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the formation

## Import
Existing formations can be imported using the combination of the application name, a colon, and the formation's type.

For example:

```
$ terraform import heroku_formation.foobar-web foobar:web
```
