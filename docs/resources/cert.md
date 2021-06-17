---
layout: "heroku"
page_title: "Heroku: heroku_cert"
sidebar_current: "docs-heroku-resource-cert"
description: |-
  Provides a Heroku SSL certificate resource. It allows to set a given certificate for a Heroku app. This resource is deprecated in favor of `heroku_ssl`.
---

# heroku\_cert

Provides a Heroku SSL certificate resource. It allows to set a given certificate for a Heroku app.This resource is deprecated in favor of `heroku_ssl`.

## Example Usage

```hcl-terraform
# Create a new Heroku app
resource "heroku_app" "default" {
  name = "test-app"
  region = "us"
}

# Add-on SSL to application
resource "heroku_addon" "ssl" {
  app  = heroku_app.default.name
  plan = "ssl"
}

# Establish certificate for a given application
resource "heroku_cert" "ssl_certificate" {
  app               = heroku_app.default.name
  certificate_chain = file("server.crt")
  private_key       = file("server.key")
  depends_on        = ["heroku_addon.ssl"]
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The Heroku app to add to.
* `certificate_chain` - (Required) The certificate chain to add
* `private_key` - (Required) The private key for a given certificate chain

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the add-on
* `cname` - The CNAME for the SSL endpoint
* `name` - The name of the SSL certificate

## Importing

When importing a Heroku cert resource, the ID must be built using the app name colon the unique ID from the Heroku API. For an app named `production-api` with a certificate ID of `b85d9224-310b-409b-891e-c903f5a40568`, you would import it as:

```
$ terraform import heroku_cert.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568
```
