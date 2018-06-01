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
data "heroku_space" "default"
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
* `shielded` - Whether or not the space has [Shield](https://devcenter.heroku.com/articles/private-spaces#shield-private-spaces) turned on. One of `on` or `off`.
* `organization` - The organization that owns this space, if the space is owned by an organization. The fields for this block are documented below.

The `organization` block supports:

* `name` (string) - The name of the organization.
