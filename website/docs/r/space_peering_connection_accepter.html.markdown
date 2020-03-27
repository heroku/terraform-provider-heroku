---
layout: "heroku"
page_title: "Heroku: heroku_space_peering_connection_accepter"
sidebar_current: "docs-heroku-resource-space-peering-connection-accepter"
description: |-
  Provides a resource for accepting VPC peering requests to Heroku Private Spaces.
---

# heroku\_space\_peering\_connection\_accepter

Provides a resource for accepting VPC peering requests to Heroku Private Spaces.

## Example Usage

```hcl
# Fetch the peering information for the Heroku Private Space.
data "heroku_space_peering_info" "peer_space" {
  name = "my-fancy-space"
}

# Initiate the request.
resource "aws_vpc_peering_connection" "request" {
  peer_owner_id = "${data.heroku_space_peering_info.peer_space.aws_account_id}"
  peer_vpc_id   = "${data.heroku_space_peering_info.peer_space.vpc_id}"
  vpc_id        = "${aws_vpc.main.id}"
}

# Accept the request.
resource "heroku_space_peering_connection_accepter" "accept" {
  space                     = "${heroku_space.peer_space.id}"
  vpc_peering_connection_id = "${aws_vpc_peering_connection.request.id}"
}
```

## Argument Reference

The following arguments are supported:

* `space` - (Required) The `UUID` of the space.
* `vpc_peering_connection_id` - (Required) The peering connection request ID.

## Attributes Reference

The following attributes are exported:

* `status` - The status of the peering connection request.
* `type` - The type of the peering connection.
