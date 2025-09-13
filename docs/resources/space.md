---
layout: "heroku"
page_title: "Heroku: heroku_space"
sidebar_current: "docs-heroku-resource-space"
description: |-
  Provides a Heroku Space resource for running apps in isolated, highly available, secure app execution environments.
---

# heroku\_space

Provides a Heroku Private Space resource for running apps in isolated, highly available, secure app execution environments.

Both generations of the Heroku platform offer Private Spaces:

* **Cedar** (default): The [Cedar generation](https://devcenter.heroku.com/articles/private-spaces#additional-features-for-cedar-private-spaces) supports all Private Space features, including Shield spaces.
* ```
* **Fir**: The next-generation platform supports enhanced capabilities for Cloud Native Buildpacks (CNB), but with has some [feature limitations](https://devcenter.heroku.com/articles/generations?preview=1#feature-parity) compared to Cedar Private Spaces.

~> **Note:** You can't change the `generation` parameter after space creation. Choose carefully based on your application requirements.

## Example Usage

A Heroku "team" was originally called an "organization", and that is still 
the identifier used in this resource.

```hcl-terraform
// Create a new Cedar-generation space (default)
resource "heroku_space" "cedar_space" {
  name = "test-cedar-space"
  organization = "my-company"
  region = "virginia"
  shield = true  // Cedar supports Shield spaces
}

// Create a new Fir-generation space
resource "heroku_space" "fir_space" {
  name = "test-fir-space"
  organization = "my-company"
  region = "virginia"
  generation = "fir"
  // Note: Shield spaces are unavailable for the Fir generation.
}

// Create a new Heroku app in test-space, same region
resource "heroku_app" "default" {
  name = "test-app"
  region = "virginia"
  space = heroku_space.cedar_space.id
  organization = {
    name = "my-company"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Private Space.
* `organization` - (Required) The name of the Heroku Team which will own the Private Space.
* `generation` - (Optional) The generation of the Heroku platform for the space. Valid values are `cedar` and `fir`. Defaults to `cedar` for backward compatibility. It can't be changed after space creation.
* `cidr` - (Optional) The RFC-1918 CIDR the Private Space will use.
  It must be a /16 in 10.0.0.0/8, 172.16.0.0/12 or 192.168.0.0/16
* `data_cidr` - (Optional) The RFC-1918 CIDR that the Private Space will use for the Heroku-managed peering connection
  that's automatically created when using Heroku Data add-ons. It must be between a /16 and a /20
* `region` - (Optional) The [region](https://devcenter.heroku.com/articles/regions#viewing-available-regions) to provision a space in.
* `shield` - (Optional) `true` if provisioning as a [Shield Private Space](https://devcenter.heroku.com/articles/private-spaces#shield-private-spaces). Note: Shield spaces are only supported for the `cedar` generation.

## Attributes Reference

The following attributes are exported:

* `id` - The ID (UUID) of the space.
* `name` - The space's name.
* `organization` - The space's Heroku Team.
* `generation` - The generation of the Heroku platform for the space (`cedar` or `fir`).
* `region` - The space's region.
* `cidr` - The space's CIDR.
* `data_cidr` - The space's Data CIDR.
* `outbound_ips` - The space's stable outbound [NAT IPs](https://devcenter.heroku.com/articles/platform-api-reference#space-network-address-translation). Note: Outbound IP management is not supported for `fir` generation spaces.

## Import

Spaces can be imported using the space `id`, e.g.

```
$ terraform import heroku_space.foobar MySpace
```
