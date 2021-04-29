package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

type spaceWithNAT struct {
	heroku.Space
	NAT heroku.SpaceNAT
}

func resourceHerokuSpace() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceCreate,
		Read:   resourceHerokuSpaceRead,
		Update: resourceHerokuSpaceUpdate,
		Delete: resourceHerokuSpaceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cidr": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "10.0.0.0/16",
				ForceNew: true,
			},

			"data_cidr": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "10.1.0.0/16",
				ForceNew: true,
			},

			"outbound_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"shield": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
		},
	}
}

func resourceHerokuSpaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.SpaceCreateOpts{}
	opts.Name = d.Get("name").(string)
	opts.Team = d.Get("organization").(string)

	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		opts.Region = &vs
	}

	if v := d.Get("shield"); v != nil {
		vs := v.(bool)
		if vs {
			log.Printf("[DEBUG] Creating a shield space")
		}
		opts.Shield = &vs
	}

	if v := d.Get("cidr"); v != nil {
		vs := v.(string)
		opts.CIDR = &vs
	}

	if v := d.Get("data_cidr"); v != nil {
		vs := v.(string)
		opts.DataCIDR = &vs
	}

	space, err := client.SpaceCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(space.ID)
	log.Printf("[INFO] Space ID: %s", d.Id())

	// Wait for the Space to be allocated
	log.Printf("[DEBUG] Waiting for Space (%s) to be allocated", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"allocating"},
		Target:  []string{"allocated"},
		Refresh: SpaceStateRefreshFunc(client, d.Id()),
		Timeout: 20 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Space (%s) to become available: %s", d.Id(), err)
	}

	config := meta.(*Config)
	time.Sleep(time.Duration(config.PostSpaceCreateDelay) * time.Second)

	return resourceHerokuSpaceRead(d, meta)
}

func resourceHerokuSpaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	spaceRaw, _, err := SpaceStateRefreshFunc(client, d.Id())()
	if err != nil {
		return err
	}

	space := spaceRaw.(*spaceWithNAT)

	d.Set("name", space.Name)
	d.Set("organization", space.Organization.Name)
	d.Set("region", space.Region.Name)
	d.Set("outbound_ips", space.NAT.Sources)
	d.Set("shield", space.Shield)
	d.Set("cidr", space.CIDR)
	d.Set("data_cidr", space.DataCIDR)

	log.Printf("[DEBUG] Set NAT source IPs to %s for %s", space.NAT.Sources, d.Id())

	return nil
}

func resourceHerokuSpaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts := heroku.SpaceUpdateOpts{Name: &name}

		_, err := client.SpaceUpdate(context.TODO(), d.Id(), opts)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceHerokuSpaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting space: %s", d.Id())
	_, err := client.SpaceDelete(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// SpaceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Space.
func SpaceStateRefreshFunc(client *heroku.Service, id string) resource.StateRefreshFunc {
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
