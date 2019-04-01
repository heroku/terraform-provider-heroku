---
layout: "heroku"
page_title: "Heroku: heroku_space"
sidebar_current: "docs-heroku-datasource-space-x"
description: |-
  Get information on a Heroku Private Space.
---

# Data Source: heroku_space

Use this data source to get information about a [Heroku Private Space](https://www.heroku.com/private-spaces).

## Example Usage

```hcl
# Look up a Heroku Private Space
data "heroku_space" "default" {
  name   = "my-secret-space"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Heroku Private Space.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the Heroku Private Space. In Heroku, this is also the unique .
* `id` - The unique ID of the Heroku Private Space.
* `region` - The region in which the Heroku Private Space is deployed.
* `state` - The state of the Heroku Private Space. Either `allocating` or `allocated`.
* `shield` - Whether or not the space has [Shield](https://devcenter.heroku.com/articles/private-spaces#shield-private-spaces) turned on. One of `on` or `off`.
* `organization` - The Heroku Team that owns this space. The fields for this block are documented below.
* `cidr` - The RFC-1918 CIDR the Private Space will use. It must be a /16 in 10.0.0.0/8, 172.16.0.0/12 or 192.168.0.0/16
* `data_cidr` - The RFC-1918 CIDR that the Private Space will use for the Heroku-managed peering connection that’s automatically created when using Heroku Data add-ons. It must be between a /16 and a /20
* `outbound_ips` - The space's stable outbound [NAT IPs](https://devcenter.heroku.com/articles/platform-api-reference#space-network-address-translation).

The `organization` block supports:

* `name` (string) - The name of the Heroku Team.
