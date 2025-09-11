---
layout: "heroku"
page_title: "Heroku: heroku_space_inbound_ruleset"
sidebar_current: "docs-heroku-resource-space-inbound-ruleset"
description: |-
  Provides a resource for managing inbound rulesets for Heroku Private Spaces.
---

# heroku\_space\_inbound\_ruleset

Provides a resource for managing [inbound rulesets](https://devcenter.heroku.com/articles/platform-api-reference#inbound-ruleset) for Heroku Private Spaces.

## Example Usage

```hcl-terraform
# Create a new Heroku space (Cedar generation - supports inbound rulesets)
resource "heroku_space" "default" {
  name         = "test-space"
  organization = "my-company"
  region       = "virginia"
  generation   = "cedar"
}

# Allow all traffic EXCEPT 8.8.4.4 to access the HPS.
resource "heroku_space_inbound_ruleset" "default" {
  space      = heroku_space.default.id
  generation = "cedar"

  rule {
    action = "allow"
    source = "0.0.0.0/0"
  }

  rule {
    action = "deny"
    source = "8.8.4.4/32"
  }
}
```

## Argument Reference

The following arguments are supported:

* `space` - (Required) The ID of the space.
* `generation` - (Optional) Generation of the space for inbound ruleset. Valid values are `cedar` and `fir`. Defaults to `cedar` for backward compatibility. **ForceNew**. Note: Inbound rulesets are not supported for `fir` generation spaces.
* `rule` - (Required) At least one `rule` block. Rules are documented below.

A `rule` block supports the following arguments:

* `action` - (Required) The action to apply this rule to. Must be one of `allow` or `deny`.
* `source` - (Required) A CIDR block source for the rule.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the inbound ruleset.
* `generation` - Generation of the space for inbound ruleset.
