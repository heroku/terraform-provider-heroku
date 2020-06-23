package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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

	var pipelineID, sourceAppName string
	var targetAppNames []string

	log.Println("[DEBUG] resourceHerokuPipelinePromotionCreate")

	if v, ok := d.GetOk("pipeline"); ok {
		pipelineID = v.(string)
		log.Printf("[DEBUG] pipeline: %v", pipelineID)
	}

	if v, ok := d.GetOk("source"); ok {
		sourceAppName = v.(string)
		log.Printf("[DEBUG] source: %q", sourceAppName)
	}

	if targets, ok := d.GetOk("targets"); ok {
		for _, v := range targets.([]interface{}) {
			t := v.(string)
			targetAppNames = append(targetAppNames, t)
		}
		log.Printf("[DEBUG] targets: %q", targetAppNames)
	}

	opts, err := createPipelinePromotionCreateOpts(pipelineID, sourceAppName, targetAppNames)
	if err != nil {
		log.Fatal("Error in create opts...")
	}

	p, err := client.PipelinePromotionCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("Error creating pipeline promotion: %s", err)
	}

	// Wait for the PipelinePromotion to be complete
	log.Printf("[INFO] Waiting for PipelinePromotion (%s) to complete", p.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"completed", "succeeded"},
		Refresh: PipelinePromotionStateRefreshFunc(client, p.ID),
		Timeout: 5 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	d.SetId(p.ID)

	log.Printf("[INFO] PipelinePromotion (%s) complete.", d.Id())

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

	// Set basic promotion info
	d.Set("pipeline", p.Pipeline.ID)
	d.Set("source", p.Source.App.ID)
	d.Set("release_id", p.Source.Release.ID)
	d.Set("status", p.Status)
	d.Set("created_at", p.CreatedAt)
	d.Set("updated_at", p.UpdatedAt)

	// Retrieve the list of promotion targets
	var pplr heroku.PipelinePromotionTargetListResult
	pplr, err = client.PipelinePromotionTargetList(context.TODO(), d.Id(), &heroku.ListRange{})
	if err != nil {
		return fmt.Errorf("Error retrieving pipeline promotion: %s", err)
	}

	// Extract the list of target app IDs
	var targets []string
	for _, v := range pplr {
		targets = append(targets, v.App.ID)
	}

	// Set the list of apps
	if err := d.Set("targets", targets); err != nil {
		return err
	}

	return nil
}

//
// Deeply nested Go structs are ridiculously hard to grok, much less build.
// Here's a trick to get most of the way to what we need. Build a JSON string
// like the following:
// {
// 	"pipeline": {
// 		"id": "abc"
// 	},
// 	"source": {
// 		"app": {
// 			"id": "def"
// 		}
// 	},
// 	"targets": [
// 		{
// 			"app": {
// 				"id": "ghi"
// 			}
// 		}
// 	]
// }
//
// ... then decode the above into a Go struct. Here's an example:
// https://play.golang.org/p/cjPbd8XifwI
//
// It's still pretty rough sledding. As a result, I've isolated it
// into a func and in that func I've broken it into chunks to make
// it a bit more digestible. I hope this helps.
//
func createPipelinePromotionCreateOpts(pipelineID, sourceApp string, targetApps []string) (heroku.PipelinePromotionCreateOpts, error) {
	// Pipeline
	pipeline := (struct {
		ID string "json:\"id\" url:\"id,key\""
	}{
		ID: pipelineID,
	})

	// Source app
	var sourceAppName *string
	sourceAppName = &sourceApp

	source := (*struct {
		ID *string "json:\"id,omitempty\" url:\"id,omitempty,key\""
	})(&struct {
		ID *string "json:\"id,omitempty\" url:\"id,omitempty,key\""
	}{
		ID: sourceAppName,
	})

	// Target apps
	type pipelinePomotionTargets []struct {
		App *struct {
			ID *string "json:\"id,omitempty\" url:\"id,omitempty,key\""
		} "json:\"app,omitempty\" url:\"app,omitempty,key\""
	}

	targets := make(pipelinePomotionTargets, len(targetApps))
	for i := 0; i < len(targetApps); i++ {
		name := targetApps[i]

		target := (*struct {
			ID *string "json:\"id,omitempty\" url:\"id,omitempty,key\""
		})(&struct {
			ID *string "json:\"id,omitempty\" url:\"id,omitempty,key\""
		}{
			ID: &name,
		})

		targets[i].App = target
	}

	// Build the create opts struct based on the parts above
	createOpts := heroku.PipelinePromotionCreateOpts{
		Pipeline: pipeline,
		Source: struct {
			App *struct {
				ID *string "json:\"id,omitempty\" url:\"id,omitempty,key\""
			} "json:\"app,omitempty\" url:\"app,omitempty,key\""
		}{
			App: source,
		},
		Targets: targets,
	}

	log.Printf("[DEBUG] CREATEOPTS: %#v", createOpts)
	log.Printf("PIPELINE ID: %s", createOpts.Pipeline.ID)
	log.Printf("SOURCE APP ID: %s", *createOpts.Source.App.ID)
	log.Printf("TARGET APP1 ID: %s", *createOpts.Targets[0].App.ID)
	log.Printf("TARGET APP2 ID: %s", *createOpts.Targets[1].App.ID)

	return createOpts, nil
}

// Returns a resource.StateRefreshFunc that is used to watch a PipelinePromotion.
func PipelinePromotionStateRefreshFunc(client *heroku.Service, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		pp, err := client.PipelinePromotionInfo(context.TODO(), id)
		if err != nil {
			log.Printf("[DEBUG] Failed to get PipelinePromotion status: %s (%s)", err, id)
			return nil, "", err
		}

		if pp.Status == "pending" {
			log.Printf("[DEBUG] PipelinePromotion pending (%s)", id)
			return &pp, pp.Status, nil
		}

		if pp.Status == "failed" {
			return nil, "", fmt.Errorf("PipelinePromotion failed (%s)", id)
		}

		return &pp, pp.Status, nil
	}
}
