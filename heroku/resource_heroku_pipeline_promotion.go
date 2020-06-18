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

	log.Println("[DEBUG] resourceHerokuPipelinePromotionCreate")

	opts := heroku.PipelinePromotionCreateOpts{}
	if v, ok := d.GetOk("pipeline"); ok {
		opts.Pipeline.ID = v.(string)
		log.Printf("[DEBUG] PipelinePromotionCreate pipeline: %v", opts.Pipeline.ID)
	}
	if v, ok := d.GetOk("source"); ok {
		switch bb := v.(interface{}).(type) {
		case string:
			var tmp *string
			x := v.(interface{}).(string)
			tmp = &x
			fmt.Printf("This is a string: %v", tmp)
			opts.Source.App.ID = tmp
			log.Printf("[DEBUG] PipelinePromotionCreate source: %v", opts.Source.App.ID)
		case float64:
			fmt.Println("this is a float")
		case bool:
			fmt.Println("this is a boolean")
		default:
			fmt.Printf("Default value is of type %v", bb)
		}
		// tmp := v.(*string)
		// opts.Source.App.ID = tmp
		// log.Printf("[DEBUG] PipelinePromotionCreate source: %v", opts.Source.App.ID)
	}
	log.Printf("[DEBUG] PipelinePromotion opts so far: %#v", opts)

	// if v, ok := d.GetOk("source"); ok {
	// 	x := fmt.Sprintf("%v", v)
	// 	log.Printf("[DEBUG] PipelinePromotionCreate source: %v", x)
	// 	opts.Source.App.ID = heroku.String(x)
	// 	log.Printf("[DEBUG] PipelinePromotionCreate source: %v", opts.Source.App.ID)
	// 	// opts.Source.App.ID = v.(*string)
	// 	// log.Printf("[DEBUG] PipelinePromotionCreate source: %s", v)
	// } else {
	// 	log.Println("[DEBUG] FAIL")
	// }

	// src := d.Get("source").(*string)
	// log.Printf("[DEBUG] PipelinePromotionCreate source: %s", src)
	// opts.Source.App.ID = src
	// // log.Printf("[DEBUG] PipelinePromotion source: %v", opts.Source.App.ID)

	// log.Printf("[DEBUG] PipelinePromotion opts so far: %#v", opts)

	// type PipelinePromotionCreateOpts struct {
	// 	Pipeline struct {
	// 		ID string `json:"id" url:"id,key"` // unique identifier of pipeline
	// 	} `json:"pipeline" url:"pipeline,key"` // pipeline involved in the promotion
	// 	Source struct {
	// 		App *struct {
	// 			ID *string `json:"id,omitempty" url:"id,omitempty,key"` // unique identifier of app
	// 		} `json:"app,omitempty" url:"app,omitempty,key"` // the app which was promoted from
	// 	} `json:"source" url:"source,key"` // the app being promoted from
	// 	Targets []struct {
	// 		App *struct {
	// 			ID *string `json:"id,omitempty" url:"id,omitempty,key"` // unique identifier of app
	// 		} `json:"app,omitempty" url:"app,omitempty,key"` // the app is being promoted to
	// 	} `json:"targets" url:"targets,key"`
	// }

	targets := d.Get("targets").([]string) // interface{}
	for _, v := range targets {
		var target struct {
			App *struct {
				ID *string `json:"id,omitempty" url:"id,omitempty,key"`
			} `json:"app,omitempty" url:"app,omitempty,key"`
		}
		target.App.ID = heroku.String(v)
		opts.Targets = append(opts.Targets, target)
		log.Printf("[DEBUG] PipelinePromotion targets: %#v", opts.Targets)
	}

	log.Printf("[DEBUG] PipelinePromotion create configuration: %v", opts)

	p, err := client.PipelinePromotionCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("Error creating pipeline promotion: %s", err)
	}

	log.Println("[DEBUG] THIS WILL PROB NEVER BE HIT!")

	d.SetId(p.ID)

	log.Printf("[INFO] PipelinePromotion ID: %s", d.Id())

	return resourceHerokuPipelinePromotionRead(d, meta)
}

// A no-op method as there is no DELETE build in Heroku Platform API.
func resourceHerokuPipelinePromotionDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] There is no DELETE for build resource so this is a no-op.")
	return nil
}

func resourceHerokuPipelinePromotionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

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
