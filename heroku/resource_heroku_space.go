package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
)

type spaceWithRanges struct {
	heroku.Space
	TrustedIPRanges []string
	NAT             heroku.SpaceNAT
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

			"trusted_ip_ranges": {
				Type:       schema.TypeSet,
				Computed:   true,
				Optional:   true,
				MinItems:   0,
				Deprecated: "This attribute is deprecated in favor of heroku_space_inbound_ruleset. Using both at the same time will likely cause unexpected behavior.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceHerokuSpaceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

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

	space, err := config.Api.SpaceCreate(context.TODO(), opts)
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
		Refresh: SpaceStateRefreshFunc(config, d.Id()),
		Timeout: 20 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Space (%s) to become available: %s", d.Id(), err)
	}

	if ranges, ok := d.GetOk("trusted_ip_ranges"); ok {
		ips := ranges.(*schema.Set)

		var rules []*struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}

		for _, r := range ips.List() {
			rules = append(rules, &struct {
				Action string `json:"action" url:"action,key"`
				Source string `json:"source" url:"source,key"`
			}{
				Action: "allow",
				Source: r.(string),
			})
		}

		opts := heroku.InboundRulesetCreateOpts{Rules: rules}
		_, err := config.Api.InboundRulesetCreate(context.TODO(), space.ID, opts)
		if err != nil {
			return fmt.Errorf("Error creating Trusted IP Ranges for Space (%s): %s", space.ID, err)
		}
		log.Printf("[DEBUG] Set Trusted IP Ranges to %s for Space %s", ips.List(), d.Id())
	}

	return resourceHerokuSpaceRead(d, meta)
}

func resourceHerokuSpaceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	spaceRaw, _, err := SpaceStateRefreshFunc(config, d.Id())()
	if err != nil {
		return err
	}

	space := spaceRaw.(*spaceWithRanges)

	d.Set("name", space.Name)
	d.Set("organization", space.Organization.Name)
	d.Set("region", space.Region.Name)
	d.Set("trusted_ip_ranges", space.TrustedIPRanges)
	d.Set("outbound_ips", space.NAT.Sources)
	d.Set("shield", space.Shield)

	log.Printf("[DEBUG] Set NAT source IPs to %s for %s", space.NAT.Sources, d.Id())

	return nil
}

func resourceHerokuSpaceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts := heroku.SpaceUpdateOpts{Name: &name}

		_, err := config.Api.SpaceUpdate(context.TODO(), d.Id(), opts)
		if err != nil {
			return err
		}
	}

	if d.HasChange("trusted_ip_ranges") {
		var rules []*struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}
		ranges := d.Get("trusted_ip_ranges").(*schema.Set)
		for _, r := range ranges.List() {
			rules = append(rules, &struct {
				Action string `json:"action" url:"action,key"`
				Source string `json:"source" url:"source,key"`
			}{
				Action: "allow",
				Source: r.(string),
			})
		}

		opts := heroku.InboundRulesetCreateOpts{Rules: rules}
		_, err := config.Api.InboundRulesetCreate(context.TODO(), d.Id(), opts)
		if err != nil {
			return fmt.Errorf("Error creating Trusted IP Ranges for Space (%s): %s", d.Id(), err)
		}

		d.Set("trusted_ip_ranges", ranges)
	}

	return nil
}

func resourceHerokuSpaceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Deleting space: %s", d.Id())
	_, err := config.Api.SpaceDelete(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// SpaceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Space.
func SpaceStateRefreshFunc(config *Config, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		space, err := config.Api.SpaceInfo(context.TODO(), id)
		if err != nil {
			log.Printf("[DEBUG] %s (%s)", err, id)
			return nil, "", err
		}

		s := spaceWithRanges{
			Space: *space,
		}

		if space.State == "allocating" {
			log.Printf("[DEBUG] Still allocating: %s (%s)", space.State, id)
			return &s, space.State, nil
		}

		ruleset, err := config.Api.InboundRulesetCurrent(context.TODO(), id)
		if err != nil {
			log.Printf("[DEBUG] %s (%s)", err, id)
			return nil, "", err
		}

		s.TrustedIPRanges = make([]string, len(ruleset.Rules))
		for i, r := range ruleset.Rules {
			s.TrustedIPRanges[i] = r.Source
		}

		nat, err := config.Api.SpaceNATInfo(context.TODO(), id)
		if err != nil {
			return nil, "", err
		}
		s.NAT = *nat

		log.Printf("[DEBUG] Outbound NAT IPs: %s (%s)", s.NAT.Sources, id)
		log.Printf("[DEBUG] Trusted IP ranges: %s (%s)", s.TrustedIPRanges, id)

		return &s, space.State, nil
	}
}
