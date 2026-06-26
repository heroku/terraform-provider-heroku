package heroku

import (
	"context"
	"log"

	heroku "github.com/heroku/heroku-go/v6"
)

// This file holds the SDKv2-free space helpers shared by the
// terraform-plugin-framework heroku_space resource and data source. They were
// extracted from the original SDKv2 resource_heroku_space.go when that resource
// was migrated to the framework.

// spaceWithNAT bundles a Heroku space with its NAT outbound-IP details.
type spaceWithNAT struct {
	heroku.Space
	NAT heroku.SpaceNAT
}

// SpaceStateRefreshFunc returns a polling function used to watch a Space. The
// returned closure has the same shape as the SDKv2 resource.StateRefreshFunc it
// replaced (func() (interface{}, string, error)), so existing framework callers
// that invoke it directly continue to work without depending on
// terraform-plugin-sdk/v2.
func SpaceStateRefreshFunc(client *heroku.Service, id string) func() (interface{}, string, error) {
	return func() (interface{}, string, error) {
		space, err := client.SpaceInfo(context.TODO(), id)
		if err != nil {
			log.Printf("[DEBUG] %s (%s)", err, id)
			return nil, "", err
		}

		s := spaceWithNAT{
			Space: *space,
		}

		if space.State == "allocating" {
			log.Printf("[DEBUG] Still allocating: %s (%s)", space.State, id)
			return &s, space.State, nil
		}

		nat, err := client.SpaceNATInfo(context.TODO(), id)
		if err != nil {
			return nil, "", err
		}
		s.NAT = *nat

		log.Printf("[DEBUG] Outbound NAT IPs: %s (%s)", s.NAT.Sources, id)

		return &s, space.State, nil
	}
}
