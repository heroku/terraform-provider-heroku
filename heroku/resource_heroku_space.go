package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type spaceWithRanges struct {
	heroku.Space
	TrustedIPRanges []string
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

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"trusted_ip_ranges": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceHerokuSpaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	opts := heroku.SpaceCreateOpts{}
	opts.Name = d.Get("name").(string)
	opts.Team = d.Get("organization").(string)

	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		opts.Region = &vs
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

	// If there are no ranges specified, default to empty list of ranges
	ranges, ok := d.GetOk("trusted_ip_ranges")
	if !ok {
		ranges = make([]interface{}, 0)
	}

	log.Printf("[DEBUG] Setting trusted_ip_ranges")
	var rules []*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}
	for _, r := range ranges.([]interface{}) {
		log.Printf("Setting range to : %s", r.(string))
		rules = append(rules, &struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}{
			Action: "allow",
			Source: r.(string),
		})
	}

	ruleOpts := heroku.InboundRulesetCreateOpts{Rules: rules}
	_, ruleErr := client.InboundRulesetCreate(context.TODO(), space.ID, ruleOpts)
	if ruleErr != nil {
		return fmt.Errorf("Error creating Trusted IP Ranges for Space (%s): %s", space.ID, ruleErr)
	}

	return resourceHerokuSpaceRead(d, meta)
}

func resourceHerokuSpaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceRaw, _, err := SpaceStateRefreshFunc(client, d.Id())()
	if err != nil {
		return err
	}

	space := spaceRaw.(*spaceWithRanges)
	setSpaceAttributes(d, space)
	return nil
}

func resourceHerokuSpaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts := heroku.SpaceUpdateOpts{Name: &name}

		space, err := client.SpaceUpdate(context.TODO(), d.Id(), opts)
		if err != nil {
			return err
		}

		// The type conversion here can be dropped when the vendored version of
		// heroku-go is updated.
		setSpaceAttributes(d, &spaceWithRanges{Space: *space})
	}

	if d.HasChange("trusted_ip_ranges") {
		var rules []*struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}
		ranges := d.Get("trusted_ip_ranges")
		for _, r := range ranges.([]interface{}) {
			rules = append(rules, &struct {
				Action string `json:"action" url:"action,key"`
				Source string `json:"source" url:"source,key"`
			}{
				Action: "allow",
				Source: r.(string),
			})
		}

		opts := heroku.InboundRulesetCreateOpts{Rules: rules}
		_, err := client.InboundRulesetCreate(context.TODO(), d.Id(), opts)
		if err != nil {
			return fmt.Errorf("Error creating Trusted IP Ranges for Space (%s): %s", d.Id(), err)
		}

		d.Set("trusted_ip_ranges", ranges)
	}

	return nil
}

func setSpaceAttributes(d *schema.ResourceData, space *spaceWithRanges) {
	d.Set("name", space.Name)
	d.Set("organization", space.Organization.Name)
	d.Set("region", space.Region.Name)
	d.Set("trusted_ip_ranges", space.TrustedIPRanges)
}

func resourceHerokuSpaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

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
			return nil, "", err
		}

		s := spaceWithRanges{
			Space: *space,
		}
		if space.State == "allocating" {
			return &s, space.State, nil
		}

		ruleset, err := client.InboundRulesetCurrent(context.TODO(), id)
		if err != nil {
			return nil, "", err
		}
		s.TrustedIPRanges = make([]string, len(ruleset.Rules))
		for i, r := range ruleset.Rules {
			s.TrustedIPRanges[i] = r.Source
		}

		return &s, space.State, nil
	}
}
