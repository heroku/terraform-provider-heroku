---
layout: "heroku"
page_title: "Heroku: heroku_app_release"
sidebar_current: "docs-heroku-resource-app-release"
description: |-
  Provides the ability to deploy a Heroku release to an application
---

# heroku\_app\_release

Provides a [Heroku App Release](https://devcenter.heroku.com/articles/platform-api-reference#release)
resource.

An app release represents a combination of code, config vars and add-ons for an app on Heroku. 

~> **NOTE:** To use this resource, you must have uploaded a slug to Heroku using [`heroku_slug`](slug.html)
or with external tooling prior to running Terraform.

~> **NOTE:** This resource is only supported for [Cedar-generation](https://devcenter.heroku.com/articles/generations#cedar) apps.

## Example Usage
```hcl-terraform
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

# Upload your slug

resource "heroku_app_release" "foobar-release" {
    app_id = heroku_app.foobar.id
    slug_id = "01234567-89ab-cdef-0123-456789abcdef"
}
```

## Argument Reference

The following arguments are supported:

* `app_id`: (Required) The Heroku app ID (not name)
* `slug_id`: The unique identifier of slug
* `description`: The description of changes in this release

## Attributes Reference

The following attributes are exported:

* `id`: The ID of the app release

## Import
Import the most recent app release using the application name.

For example:
```
$ terraform import heroku_app_release.foobar-release foobar
```
