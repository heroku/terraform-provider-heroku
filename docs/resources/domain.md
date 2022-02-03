---
layout: "heroku"
page_title: "Heroku: heroku_domain"
sidebar_current: "docs-heroku-resource-domain"
description: |-
  Provides a Heroku App resource. This can be used to create and manage applications on Heroku.
---

# heroku\_domain

Provides a Heroku Domain resource. This can be used to
create and manage custom domains on Heroku.

## Example Usage

```hcl-terraform
# Create a new Heroku app
resource "heroku_app" "default" {
  name = "test-app"
  region = "us"
}

# Associate a custom domain
resource "heroku_domain" "default" {
  app_id   = heroku_app.default.id
  hostname = "terraform.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `hostname` - (Required) The hostname to serve requests from.
* `app_id` - (Required) Heroku app ID (do not use app name)

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the domain record.
* `hostname` - The hostname traffic will be served as.
* `cname` - The CNAME traffic should route to.
* `sni_endpoint_id` - The ID of the heroku_ssl resource to associate the domain with.

## Importing

When importing a Heroku domain resource, the ID is specified `APP_NAME:DOMAIN_IDENTIFIER`, where the domain can be identified either with the UUID from the Heroku API or the domain name.

For an app named `test-app` with a domain name of `terraform.example.com`, you could import it with:

```
$ terraform import heroku_domain.default test-app:terraform.example.com
```
