---
layout: "heroku"
page_title: "Heroku: heroku_app_release"
sidebar_current: "docs-heroku-resource-app-release"
description: |-
  Provides the ability to deploy a heroku release to an application
---

# heroku\_app\_release

Provides a [Heroku App Release](https://devcenter.heroku.com/articles/platform-api-reference#release)
resource.

An app release represents a combination of code, config vars and add-ons for an app on Heroku.

~> **NOTE:**
This resource requires the slug be uploaded to Heroku using [`heroku_slug`](slug.html)
or with external tooling prior to running terraform.

## Example Usage
```hcl-terraform
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

# Upload your slug

resource "heroku_app_release" "foobar-release" {
    app = heroku_app.foobar.id
    slug_id = "01234567-89ab-cdef-0123-456789abcdef"
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) Heroku app ID (do not use app name)
* `slug_id` - unique identifier of slug
* `description` - description of changes in this release

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the app release

## Import
Existing app releases can be imported using the combination of the application name, a colon, and the formation's type.

For example:
```
$ terraform import heroku_app_release.foobar-release foobar
```
