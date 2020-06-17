package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuPipelinePromotion() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuPipelinePromotionCreate,
		Read:   resourceHerokuPipelinePromotionRead,
		Delete: resourceHerokuPipelinePromotionDelete,

		Schema: map[string]*schema.Schema{
			"pipeline": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"targets": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"release_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuPipelinePromotionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.PipelinePromotionCreateOpts{}
	opts.Pipeline.ID = d.Get("pipeline").(string)
	opts.Source.App.ID = d.Get("source").(*string)

	targets := d.Get("targets").([]string) // interface{}
	for _, v := range targets {
		var target struct {
			App *struct {
				ID *string `json:"id,omitempty" url:"id,omitempty,key"`
			} `json:"app,omitempty" url:"app,omitempty,key"`
		}
		target.App.ID = heroku.String(v)
		opts.Targets = append(opts.Targets, target)
	}

	log.Printf("[DEBUG] PipelinePromote create configuration: %v", opts)

	p, err := client.PipelinePromotionCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("Error creating pipeline promotion: %s", err)
	}

	d.SetId(p.ID)

	log.Printf("[INFO] PipelinePromotion ID: %s", d.Id())

	// log.Printf("[INFO] PipelinePromotion succeeded: %q", p)

	return resourceHerokuPipelinePromotionRead(d, meta)
}

// A no-op method as there is no DELETE build in Heroku Platform API.
func resourceHerokuPipelinePromotionDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] There is no DELETE for build resource so this is a no-op.")
	return nil
}

func resourceHerokuPipelinePromotionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Get the basic Pipeliine Promotion info to start
	p, err := client.PipelinePromotionInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving pipeline promotion: %s", err)
	}

	d.Set("pipeline", p.Pipeline.ID)
	d.Set("source", p.Source.App.ID)
	d.Set("release_id", p.Source.Release.ID)
	d.Set("status", p.Status)
	d.Set("created_at", p.CreatedAt)
	d.Set("updated_at", p.UpdatedAt)

	// // Retrieve the list of promotion targets
	// var pplr heroku.PipelinePromotionTargetListResult
	// pplr, err = client.PipelinePromotionTargetList(context.TODO(), d.Id(), &heroku.ListRange{})
	// if err != nil {
	// 	return fmt.Errorf("Error retrieving pipeline promotion: %s", err)
	// }

	// // TODO: Not sure if a simple assignment will work here; VERIFY.
	// d.Set("targets", pplr)

	return nil
}
