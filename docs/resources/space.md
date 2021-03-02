---
layout: "heroku"
page_title: "Heroku: heroku_space"
sidebar_current: "docs-heroku-resource-space"
description: |-
  Provides a Heroku Space resource for running apps in isolated, highly available, secure app execution environments.
---

# heroku\_space

Provides a Heroku Private Space resource for running apps in isolated, highly available, secure app execution environments.

## Example Usage

A Heroku "team" was originally called an "organization", and that is still 
the identifier used in this resource.

```hcl-terraform
// Create a new Heroku space
resource "heroku_space" "default" {
  name = "test-space"
  organization = "my-company"
  region = "virginia"
}

// Create a new Heroku app in test-space
resource "heroku_app" "default" {
  name = "test-app"
  region = "us"
  space = heroku_space.default.name
  organization = {
    name = "my-company"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Private Space.
* `organization` - (Required) The name of the Heroku Team which will own the Private Space.
* `cidr` - (Optional) The RFC-1918 CIDR the Private Space will use.
  It must be a /16 in 10.0.0.0/8, 172.16.0.0/12 or 192.168.0.0/16
* `data_cidr` - (Optional) The RFC-1918 CIDR that the Private Space will use for the Heroku-managed peering connection
  that’s automatically created when using Heroku Data add-ons. It must be between a /16 and a /20
* `region` - (Optional) provision in a specific [Private Spaces region](https://devcenter.heroku.com/articles/regions#viewing-available-regions).
* `shield` - (Optional) provision as a [Shield Private Space](https://devcenter.heroku.com/articles/private-spaces#shield-private-spaces).

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the space.
* `name` - The space's name.
* `organization` - The space's Heroku Team.
* `region` - The space's region.
* `cidr` - The space's CIDR.
* `data_cidr` - The space's Data CIDR.
* `outbound_ips` - The space's stable outbound [NAT IPs](https://devcenter.heroku.com/articles/platform-api-reference#space-network-address-translation).

## Import

Spaces can be imported using the space `id`, e.g.

```
$ terraform import heroku_space.foobar MySpace
```
