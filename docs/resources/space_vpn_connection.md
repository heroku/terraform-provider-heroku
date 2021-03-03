---
layout: "heroku"
page_title: "Heroku: heroku_space_vpn_connection"
sidebar_current: "docs-heroku-resource-vpn-connection-x"
description: |-
  Create a VPN connection between a network and a Heroku Private Space.
---

# heroku\_space\_vpn\_connection

Provides a resource for creating a VPN connection between a network and a Heroku Private Space. For more information, see [Private Spaces VPN Connection](https://devcenter.heroku.com/articles/private-space-vpn-connection?preview=1) in the Heroku DevCenter.

## Example Usage

```hcl-terraform
// Create a new Heroku space
resource "heroku_space" "default" {
  name         = "test-space"
  organization = "my-company"
  region       = "virginia"
}

// Connect the Heroku space to another network with a VPN
resource "heroku_space_vpn_connection" "office" {
  name           = "office"
  space          = heroku_space.default.id
  public_ip      = "203.0.113.1"
  routable_cidrs = ["192.168.1.0/24"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the VPN connection.
* `space` - (Required) The `UUID` of the Heroku Private Space where the VPN connection will be established.
* `public_ip` - (Required) The public IP address of the VPN endpoint on the network where the VPN connection will be established.
* `routable_cidrs` - (Required) A list of IPv4 CIDR blocks used by the network where the VPN connection will be established.

## Attributes Reference

The following attributes are exported:

* `space_cidr_block` - The CIDR block for the Heroku Private Space. The network where the VPN will be established should be configured to route traffic destined for this CIDR block over the VPN link.
* `ike_version` - The IKE version used to setup the IPsec tunnel.
* `tunnels` - Details about each VPN tunnel endpoint.
  * `ip` - The public IP address of the tunnel.
  * `pre_shared_key` - The pre-shared IPSec secret for the tunnel.
