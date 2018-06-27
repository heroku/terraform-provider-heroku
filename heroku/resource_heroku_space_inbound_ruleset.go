package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/validators"
)

type spacePeerInfo struct {
	heroku.Peering
}

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
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeString,
							Required: true,
							Default:  codebuild.CacheTypeNoCache,
							ValidateFunc: validation.StringInSlice([]string{
								"allow",
								"deny",
							}, false),
						},
						"source": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateCIDRNetworkAddress,
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
		data := config.(map[string]interface{})

		ruleset = append(ruleset, &struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}{
			Action: data["action"].(string),
			Source: data["source"].(string),
		})
	}

	return heroku.InboundRulesetCreateOpts{Rules: rules}
}

func resourceHerokuSpaceInboundRulesetSet(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)
	ruleset := getRulesetFromSchema(d)

	_, err := client.InboundRulesetCreate(context.TODO(), spaceIdentity, ruleset)
	if err != nil {
		return fmt.Errorf("Error creating inbound ruleset for space (%s): %s", space.ID, err)
	}

	return resourceHerokuSpaceInboundRulesetRead(d, meta)
}

func resourceHerokuSpaceInboundRulesetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)
	ruleset, err := client.InboundRulesetCurrent(context.TODO(), spaceIdentity)
	if err != nil {
		return fmt.Errorf("Error creating inbound ruleset for space (%s): %s", space.ID, err)
	}

	rulesList := []interface{}{}
	for rule := range ruleset.Rules {
		values := map[string]interface{}{}
		values["source"] = rule.Source
		values["action"] = rule.Action
		rulesList = append(rulesList, values)
	}

	d.Set("rule", rulesList)

	return nil
}

func resourceHerokuSpaceInboundRulesetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	ruleset := &heroku.InboundRulesetCreateOpts{
		Rules: []*struct {
			Action string
			Source string
		}{
			Action: "allow",
			Source: "0.0.0.0/0",
		},
	}

	_, err := client.InboundRulesetCreate(context.TODO(), spaceIdentity, ruleset)
	if err != nil {
		return fmt.Errorf("Error resettting inbound ruleset for space (%s): %s", space.ID, err)
	}

	d.SetId("")
	return nil
}
