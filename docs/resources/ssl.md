---
layout: "heroku"
page_title: "Heroku: heroku_ssl"
sidebar_current: "docs-heroku-resource-ssl"
description: |-
  Provides a Heroku SSL certificate resource. It allows to set a given certificate for a domain on a Heroku app.
---

# heroku\_ssl

Provides a Heroku SSL certificate resource. It allows to set a given certificate for a domain on a Heroku app.

## Example Usage

```hcl-terraform
# Create a new Heroku app
resource "heroku_app" "default" {
  name = "test-app"
  region = "us"
}

# Build a slug that we can scale
resource "heroku_build" "default" {
  app = heroku_app.default.name
  source {
    url = "https://github.com/heroku/terraform-provider-heroku/raw/master/heroku/test-fixtures/app.tgz"
  }
}

# Scale the app to a tier that supports Heroku SSL
resource "heroku_formation" "web" {
  app = heroku_app.default.name
  type = "web"
  size = "hobby"
  quantity = 1
  # Wait until the build has completed before attempting to scale
  depends_on = [heroku_build.default]
}

# Create the certificate
resource "heroku_ssl" "one" {
  app = heroku_app.default.name
  certificate_chain = file("server.crt")
  private_key = file("server.key")
  # Wait until the process_tier changes to hobby before attempting to create a cert
  depends_on = [heroku_formation.web]
}

# Associate it with a domain
resource "heroku_domain" "one" {
  app = heroku_app.default.name
  hostname = "terraform-123.example.com"
  sni_endpoint_id = heroku_ssl.one.id
}

# Create another certificate
resource "heroku_ssl" "two" {
  app = heroku_app.default.name
  certificate_chain = file("server.crt")
  private_key = file("server.key")
  # Wait until the process_tier changes to hobby before attempting to create a cert
  depends_on = [heroku_formation.web]
}

# Associate it with a second domain
resource "heroku_domain" "two" {
  app = heroku_app.default.name
  hostname = "terraform-456.example.com"
  sni_endpoint_id = heroku_ssl.two.id
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The Heroku app to add to.
* `certificate_chain` - (Required) The certificate chain to add
* `private_key` - (Required) The private key for a given certificate chain

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the SSL certificate
* `name` - The name of the SSL certificate

## Importing

When importing a Heroku ssl resource, the ID must be built using the app name colon the unique ID from the Heroku API. For an app named `production-api` with a certificate ID of `b85d9224-310b-409b-891e-c903f5a40568`, you would import it as:

```
$ terraform import heroku_ssl.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568
```
