package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuSpaceInboundRuleset() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceInboundRulesetSet,
		Read:   resourceHerokuSpaceInboundRulesetRead,
		Update: resourceHerokuSpaceInboundRulesetSet,
		Delete: resourceHerokuSpaceInboundRulesetDelete,

		Schema: map[string]*schema.Schema{
			"space": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rule": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"source": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsCIDRNetwork(0, 32),
						},
					},
				},
			},
		},
	}
}

func getRulesetFromSchema(d *schema.ResourceData) heroku.InboundRulesetCreateOpts {
	rules := d.Get("rule").(*schema.Set)

	var ruleset []*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}

	for _, r := range rules.List() {
		data := r.(map[string]interface{})

		ruleset = append(ruleset, &struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}{
			Action: data["action"].(string),
			Source: data["source"].(string),
		})
	}

	return heroku.InboundRulesetCreateOpts{Rules: ruleset}
}

func resourceHerokuSpaceInboundRulesetSet(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	spaceIdentity := d.Get("space").(string)
	ruleset := getRulesetFromSchema(d)

	_, err := client.InboundRulesetCreate(context.TODO(), spaceIdentity, ruleset)
	if err != nil {
		return fmt.Errorf("Error creating inbound ruleset for space (%s): %s", spaceIdentity, err)
	}

	return resourceHerokuSpaceInboundRulesetRead(d, meta)
}

func resourceHerokuSpaceInboundRulesetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	spaceIdentity := d.Get("space").(string)
	ruleset, err := client.InboundRulesetCurrent(context.TODO(), spaceIdentity)
	if err != nil {
		return fmt.Errorf("Error creating inbound ruleset for space (%s): %s", spaceIdentity, err)
	}

	rulesList := []interface{}{}
	for _, rule := range ruleset.Rules {
		values := map[string]interface{}{}
		values["source"] = rule.Source
		values["action"] = rule.Action
		rulesList = append(rulesList, values)
	}

	d.SetId(ruleset.ID)
	d.Set("rule", rulesList)
	d.Set("space", ruleset.Space.Name)

	return nil
}

func resourceHerokuSpaceInboundRulesetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	spaceIdentity := d.Get("space").(string)

	// Heroku Private Spaces ship with a default 0.0.0.0/0 inbound ruleset. An HPS *MUST* have
	// an inbound ruleset. There's no delete API method for this. So when we "delete" the ruleset
	// we reset things back to what Heroku sets when the HPS is created. Given that the default
	// allows all traffic from all places, this is akin to deleting all filtering.
	var rules []*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}

	rules = append(rules, &struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}{
		Action: "allow",
		Source: "0.0.0.0/0",
	})

	_, err := client.InboundRulesetCreate(context.TODO(), spaceIdentity, heroku.InboundRulesetCreateOpts{Rules: rules})
	if err != nil {
		return fmt.Errorf("Error resetting inbound ruleset for space (%s): %s", spaceIdentity, err)
	}

	d.SetId("")
	return nil
}
