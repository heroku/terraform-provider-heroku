---
layout: "heroku"
page_title: "Heroku: heroku_ssl"
sidebar_current: "docs-heroku-resource-ssl"
description: |-
  Provides a Heroku SSL certificate resource to manage a certificate for a Heroku app.
---

# heroku\_ssl

This resource manages an SSL certificate for a Heroku app.

-> **IMPORTANT!**
This resource renders the "private_key" attribute in plain-text in your state file.
Please ensure that your state file is properly secured and encrypted at rest.

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
  app_id = heroku_app.default.uuid
  certificate_chain = file("server.crt")
  private_key = file("server.key")

  # Wait until the process_tier changes to hobby before attempting to create a cert
  depends_on = [heroku_formation.web]
}

resource "heroku_domain" "no-ssl" {
  app = heroku_app.default.name
  hostname = "terraform-123-no-ssl.example.com"
  # Until November 2021 if you have an ssl resource, but do not want to associate it with a domain, you must ensure the domain is created after the ssl resource. See https://devcenter.heroku.com/changelog-items/2192 for more details. We do this by adding a depends_on for the ssl resources.
  depends_on = [heroku_ssl.one]
}

# Associate it with a domain
resource "heroku_domain" "one" {
  app = heroku_app.default.name
  hostname = "terraform-123.example.com"
  sni_endpoint_id = heroku_ssl.one.id
}

# Create another certificate
resource "heroku_ssl" "two" {
  app_id = heroku_app.default.uuid
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

* `app_id` - (Required) The Heroku app UUID to add to.
* `certificate_chain` - (Required) The certificate chain to add.
* `private_key` - (Optional) The private key for a given certificate chain. You **must** set this attribute when creating or
  updating an SSL resource. However, **do not** set a value for this attribute if you are initially importing an existing
  SSL resource. The attribute value does not get displayed in logs or regular output.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the SSL certificate
* `name` - The name of the SSL certificate

## Importing

An existing SSL resource can be imported using a composite value of the app name and certificate UUID separated by a colon.

For example:

```
$ terraform import heroku_ssl.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568
```
