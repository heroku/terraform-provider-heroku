---
layout: "heroku"
page_title: "Heroku: heroku_drain"
sidebar_current: "docs-heroku-resource-drain"
description: |-
  Provides a Heroku Drain resource. This can be used to create and manage Log Drains on Heroku.
---

# heroku\_drain

Provides a Heroku Drain resource. This can be used to
create and manage Log Drains on Heroku.

## Example Usage

```hcl-terraform
resource "heroku_app" "foobar" {
  name = "foobar"
  region = "us"
}

resource "heroku_drain" "default" {
  app = heroku_app.foobar.id
  url = "syslog://terraform.example.com:1234"
}
```

## Argument Reference

The following arguments are supported:

* `url` - (Required) The URL for Heroku to drain your logs to.
* `app` - (Required) The Heroku app to link to.

## Attributes Reference

The following attributes are exported:

* `token` - The unique token for your created drain.

## Importing

When importing a Heroku drain resource, the ID must be built using the app name colon the unique ID from the Heroku API. For an app named `production-api` with a drain ID of `b85d9224-310b-409b-891e-c903f5a40568`, you would import it as: 

```
$ terraform import heroku_drain.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568
```
