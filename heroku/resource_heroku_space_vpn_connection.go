package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuSpaceVPNConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceVPNConnectionCreate,
		Read:   resourceHerokuSpaceVPNConnectionRead,
		Delete: resourceHerokuSpaceVPNConnectionDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"space": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public_ip": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"routable_cidrs": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
			},

			"space_cidr_block": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ike_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"tunnels": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"pre_shared_key": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
					},
				},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
		},
	}
}

func resourceHerokuSpaceVPNConnectionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	space, id, err := parseCompositeID(d.Id())
	if err != nil {
		return err
	}

	conn, err := client.VPNConnectionInfo(context.TODO(), space, id)
	if err != nil {
		return fmt.Errorf("Error reading VPN information: %v", err)
	}

	d.Set("space", space)
	d.Set("name", conn.Name)
	d.Set("public_ip", conn.PublicIP)
	d.Set("routable_cidrs", conn.RoutableCidrs)
	d.Set("space_cidr_block", conn.SpaceCIDRBlock)
	d.Set("ike_version", conn.IKEVersion)

	tunnels := []map[string]interface{}{}
	for _, t := range conn.Tunnels {
		tunnels = append(tunnels, map[string]interface{}{
			"ip":             t.IP,
			"pre_shared_key": t.PreSharedKey,
		})
	}
	d.Set("tunnels", tunnels)

	return nil
}

func resourceHerokuSpaceVPNConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	space := d.Get("space").(string)

	routableCIDRs := []string{}
	for _, v := range d.Get("routable_cidrs").(*schema.Set).List() {
		routableCIDRs = append(routableCIDRs, v.(string))
	}

	conn, err := client.VPNConnectionCreate(context.TODO(), space, heroku.VPNConnectionCreateOpts{
		Name:          d.Get("name").(string),
		PublicIP:      d.Get("public_ip").(string),
		RoutableCidrs: routableCIDRs,
	})
	if err != nil {
		return fmt.Errorf("Error creating VPN: %v", err)
	}

	log.Printf("[DEBUG] Waiting for VPN (%s) to be allocated", conn.ID)
	stateConf := &resource.StateChangeConf{
		Pending:      []string{"pending", "provisioning"},
		Target:       []string{"active"},
		Refresh:      spaceVPNConnectionStateRefreshFunc(client, space, conn.ID),
		Timeout:      30 * time.Minute,
		PollInterval: 20 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("error waiting for VPN to become available: %s", err)
	}

	d.SetId(buildCompositeID(space, conn.ID))

	return resourceHerokuSpaceVPNConnectionRead(d, meta)
}

func resourceHerokuSpaceVPNConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	space, id, err := parseCompositeID(d.Id())
	if err != nil {
		return err
	}

	_, err = client.VPNConnectionDestroy(context.TODO(), space, id)
	if err != nil {
		return fmt.Errorf("Error deleting VPN: %v", err)
	}

	d.SetId("")
	return nil
}

func spaceVPNConnectionStateRefreshFunc(client *heroku.Service, space, connectionID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		vpn, vpnGetErr := client.VPNConnectionInfo(context.TODO(), space, connectionID)

		// Retry on "not found"
		if vpnGetErr != nil && strings.Contains(vpnGetErr.Error(), "VPN is not found") {
			return vpn, "pending", nil
		}

		// Fail for any remaining error
		if vpnGetErr != nil {
			return nil, "failed", fmt.Errorf("error fetching VPN connection status: %s", vpnGetErr)
		}

		if vpn.Status != "active" {
			return vpn, vpn.Status, nil
		}

		return vpn, vpn.Status, nil
	}
}
