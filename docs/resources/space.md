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
* **Fir**: The next-generation platform supports enhanced capabilities for Cloud Native Buildpacks (CNB), but with has some [feature limitations](https://devcenter.heroku.com/articles/generations#feature-parity) compared to Cedar Private Spaces.

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

* `name` - (Required) The name of the space.
* `organization` - (Required) The name of the Heroku team to designate as owner of the space.
* `generation` - (Optional) The generation of the Heroku platform for the space ( `cedar` or `fir`). Defaults to `cedar` for backward compatibility. It can't be changed after space creation.
* `cidr` - (Optional) The RFC-1918 CIDR block for the space to use.
  It must be a `/16` subnet in `10.0.0.0/8`, `172.16.0.0/12` or `192.168.0.0/16`
* `data_cidr` - (Optional) The RFC-1918 CIDR block for the Private Space to use for the Heroku-managed peering connection
  that's automatically created when using Heroku Data add-ons. It must be between a `/16` and a `/20` subnet.
* `region` - (Optional) The [region](https://devcenter.heroku.com/articles/regions#viewing-available-regions) to provision the space in.
* `shield` - (Optional) `true` if provisioning as a [Shield Private Space](https://devcenter.heroku.com/articles/private-spaces#shield-private-spaces). **Note:** Shield spaces are only supported for the `cedar` generation.

## Attributes Reference

The following attributes are exported:

* `id` - The ID (UUID) of the space.
* `name` - The name of the space.
* `organization` - The name of Heroku team that owns the space.
* `generation` - The generation of the Heroku platform for the space (`cedar` or `fir`).
* `region` - The region the space is in.
* `cidr` - The CIDR block for the space.
* `data_cidr` - The data CIDR for the space.
* `outbound_ips` - The stable outbound [NAT IPs](https://devcenter.heroku.com/articles/platform-api-reference#space-network-address-translation) of the space. **Note**: Outbound IP management is only supported for the `cedar` generation.

## Import

Import a space using the space `id`.

For example:

```
$ terraform import heroku_space.foobar MySpace
```
