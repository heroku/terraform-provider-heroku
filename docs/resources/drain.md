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
  app_id = heroku_app.foobar.id
  url = "syslog://terraform.example.com:1234"
}
```

```hcl-terraform
resource "heroku_app" "foobar" {
  name = "foobar"
  region = "us"
}

resource "heroku_drain" "default" {
  app_id = heroku_app.foobar.id
  sensitive_url = "https://user:pass@terraform.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `app_id` - (Required) Heroku app ID (do not use app name)
* `url` - (Optional) The URL for Heroku to drain your logs to. Either `url` or `sensitive_url` must be defined.
* `sensitive_url` - (Optional) The URL for Heroku to drain your logs to. The main difference between `sensitive_url` and `url`
is `sensitive_url` outputs are redacted, with <sensitive> displayed in place of their value during a `terraform apply`
or `terraform refresh`. Either `url` or `sensitive_url` must be defined.

## Attributes Reference

The following attributes are exported:

* `token` - The unique token for your created drain.

## Importing

When importing a Heroku drain resource, the ID must be built using the app name colon the unique ID from the Heroku API.
For an app named `production-api` with a drain ID of `b85d9224-310b-409b-891e-c903f5a40568` and the `url` attribute value
defined for the resource, you would import it as:

```
$ terraform import heroku_drain.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568
```

When importing a Heroku drain resource, the ID must be built using the app name colon the unique ID from the Heroku API.
For an app named `production-api` with a drain ID of `b85d9224-310b-409b-891e-c903f5a40568` and the `sensitive_url` attribute value
defined for the resource, you would import it as:

```
$ terraform import heroku_drain.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568:sensitive
```