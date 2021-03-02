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
  app      = heroku_app.default.name
  hostname = "terraform.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `hostname` - (Required) The hostname to serve requests from.
* `app` - (Required) The Heroku app to link to.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the domain record.
* `hostname` - The hostname traffic will be served as.
* `cname` - The CNAME traffic should route to.

## Importing

When importing a Heroku domain resource, the ID must be built using the app name colon the unique ID from the Heroku API. For an app named `production-api` with a domain ID of `b85d9224-310b-409b-891e-c903f5a40568`, you would import it as: 

```
$ terraform import heroku_domain.production_api production-api:b85d9224-310b-409b-891e-c903f5a40568
```
