---
layout: "heroku"
page_title: "Heroku: heroku_app_webhook"
sidebar_current: "docs-heroku-resource-app-webhook"
description: |-
  Provides the ability to manage application webhooks
---

# heroku\_app\_webhook

Provides a [Heroku App Webhook](https://devcenter.heroku.com/categories/app-webhooks).

## Example Usage

```hcl
# Create a new Heroku app
resource "heroku_app" "foobar" {
  name = "foobar"
}

# Add a web-hook for the app
resource "heroku_app_webhook" "foobar_release" {
  app_id  = "${heroku_app.foobar.id}"
  level   = "notify"
  url     = "https://example.com/heroku_webhook"
  include = ["api:release"]
}
```

## Argument Reference

The following arguments are supported:

* `app_id` - (Required) A Heroku app's `UUID`. Can also be the name of the Heroku app but `UUID` is preferred as it is idempotent.
* `level` - (Required) The webhook level (either `notify` or `sync`)
* `url` - (Required) Optional plan configuration.
* `include` - (Required) List of events to deliver to the webhook.
* `secret` - (Optional) Value used to sign webhook payloads. Once set, this value cannot be fetched from the Heroku API, but it can be updated.
* `authorization` - (Optional) Values used in `Authorization` header. Once set, this value cannot be fetched from the Heroku API, but it can be updated.

## Importing

Existing webhooks can be imported using the combination of the application name or id, a colon, and the webhook name or id, e.g.

```
$ terraform import heroku_app_webhook.foobar_release foobar:b85d9224-310b-409b-891e-c903f5a40568
```
