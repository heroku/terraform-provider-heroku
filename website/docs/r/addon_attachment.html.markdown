---
layout: "heroku"
page_title: "Heroku: heroku_addon_attachment"
sidebar_current: "docs-heroku-resource-addon-attachment"
description: |-
  Attaches a Heroku Addon Resource to an additional Heroku App
---

# heroku\_addon\_attachment

Attaches a Heroku Addon Resource to an additional Heroku App.

## Example Usage

```hcl
resource "heroku_addon_attachment" "database" {
  app_id  = heroku_app.default.id
  addon_id = heroku_addon.database.id
}

// attach postgres credentials
resource "heroku_addon_attachment" "database_credentials" {
  app_id  = heroku_app.default.id
  addon_id = heroku_addon.database.id
  namespace = "credential: ${var.credential_name}"
}

```

## Argument Reference

The following arguments are supported:

* `app_id` - (Required) The ID of the Heroku App to attach to.
* `addon_id` - (Required) The ID of the existing Heroku Addon to attach.
* `name` - (Optional) A friendly name for the Heroku Addon Attachment.
* `namespace` - (Optional) The namespace value for the Heroku Addon Attachment. This can be used to configure the behaviour of the attachment. See [Heroku Platform API Reference](https://devcenter.heroku.com/articles/platform-api-reference#add-on-attachment-create)

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the add-on attachment

## Import

Addons can be imported using the unique Addon Attachment `id`, e.g.

```
$ terraform import heroku_addon_attachment.foobar 01234567-89ab-cdef-0123-456789abcdef
```
