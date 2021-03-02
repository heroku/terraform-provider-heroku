---
layout: "heroku"
page_title: "Heroku: heroku_team"
sidebar_current: "docs-heroku-datasource-team-x"
description: |-
  Get information on a Heroku Team.
---

# Data Source: heroku_team

Use this data source to get information about a Heroku Team or Heroku Enterprise team.

## Example Usage

```hcl-terraform
data "heroku_team" "my_heroku_team" {
  name = "name_of_my_heroku_team"
}

output "heroku_team_data_basic" {
  value = [
    "Heroku team",
    "id: ${data.heroku_team.my_heroku_team.id}",
    "default: ${data.heroku_team.my_heroku_team.default}",
    "membership_limit: ${data.heroku_team.my_heroku_team.membership_limit}",
    "provisioned_licenses: ${data.heroku_team.my_heroku_team.provisioned_licenses}",
    "type: ${data.heroku_team.my_heroku_team.type}",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The team name

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the team

* `default` - Whether to use this team when none is specified

* `credit_card_collections` - Whether charges incurred by the team are paid by credit card

* `membership_limit` - Upper limit of members allowed in a team

* `provisioned_licenses` - Whether the team is provisioned licenses by Salesforce

* `type` - type of team Will likely be either "enterprise" or "team"
