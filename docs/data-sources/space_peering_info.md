---
layout: "heroku"
page_title: "Heroku: heroku_space_peering_info"
sidebar_current: "docs-heroku-datasource-space-peering-info-x"
description: |-
  Get peering information on a Heroku Private Space.
---

# Data Source: heroku_space_peering_info

Use this data source to get peering information about a [Heroku Private Space](https://www.heroku.com/private-spaces).

## Example Usage

```hcl-terraform
# Look up a Heroku Private Space's peering info. 
data "heroku_space_peering_info" "default" {
  name   = "my-secret-space"
}

# Initiate a VPC peering connection request.
resource "aws_vpc_peering_connection" "foo" {
  peer_owner_id = data.heroku_space_peering_info.default.aws_account_id
  peer_vpc_id   = data.heroku_space_peering_info.default.vpc_id
  vpc_id        = aws_vpc.foo.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Heroku Private Space.

## Attributes Reference

The following attributes are exported:

* `aws_account_id` - The AWS account ID that the Heroku Private Space runs in.

* `aws_region` - The AWS region that the Heroku Private Space runs in.

* `vpc_id` - The VPC ID of the Heroku Private Space.

* `vpc_cidr` - The CIDR block of the VPC ID.

* `dyno_cidr_blocks` - The CIDR blocks that the Dynos run on.

* `unavailable_cidr_blocks` - A list of unavailable CIDR blocks.
