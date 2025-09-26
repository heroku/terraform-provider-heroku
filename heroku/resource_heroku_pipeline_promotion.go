// Pipeline Promotion Resource
//
// This resource allows promoting releases between apps in a Heroku Pipeline.
// Supports promoting either the latest release or a specific release by ID.
package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v6"
)

func resourceHerokuPipelinePromotion() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuPipelinePromotionCreate,
		Read:   resourceHerokuPipelinePromotionRead,
		Delete: resourceHerokuPipelinePromotionDelete,

		Schema: map[string]*schema.Schema{
			"pipeline": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Pipeline ID for the promotion",
			},

			"source_app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Source app ID to promote from",
			},

			"release_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Release ID to promote to target apps",
			},

			"targets": {
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Description: "Set of target app IDs to promote to",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},

			// Computed fields
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the promotion (pending, completed)",
			},

			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the promotion was created",
			},

			"promoted_release_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the release that was actually promoted",
			},
		},
	}
}

func resourceHerokuPipelinePromotionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[DEBUG] Creating pipeline promotion")

	// Build promotion options with release_id support
	pipelineID := d.Get("pipeline").(string)
	sourceAppID := d.Get("source_app_id").(string)
	targets := d.Get("targets").(*schema.Set)

	opts := heroku.PipelinePromotionCreateOpts{}
	opts.Pipeline.ID = pipelineID
	opts.Source.App = &struct {
		ID *string `json:"id,omitempty" url:"id,omitempty,key"`
	}{ID: &sourceAppID}

	// Set required release_id
	releaseIDStr := d.Get("release_id").(string)
	opts.Source.Release = &struct {
		ID *string `json:"id,omitempty" url:"id,omitempty,key"`
	}{ID: &releaseIDStr}
	log.Printf("[DEBUG] Promoting release: %s", releaseIDStr)

	// Convert targets set to slice
	for _, target := range targets.List() {
		targetAppID := target.(string)
		targetApp := &struct {
			ID *string `json:"id,omitempty" url:"id,omitempty,key"`
		}{ID: &targetAppID}

		opts.Targets = append(opts.Targets, struct {
			App *struct {
				ID *string `json:"id,omitempty" url:"id,omitempty,key"`
			} `json:"app,omitempty" url:"app,omitempty,key"`
		}{App: targetApp})
	}

	log.Printf("[DEBUG] Pipeline promotion create configuration: %#v", opts)

	promotion, err := client.PipelinePromotionCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("error creating pipeline promotion: %s", err)
	}

	log.Printf("[INFO] Created pipeline promotion ID: %s", promotion.ID)
	d.SetId(promotion.ID)

	return resourceHerokuPipelinePromotionRead(d, meta)
}

func resourceHerokuPipelinePromotionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[DEBUG] Reading pipeline promotion: %s", d.Id())

	promotion, err := client.PipelinePromotionInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("error retrieving pipeline promotion: %s", err)
	}

	// Set computed fields
	d.Set("status", promotion.Status)
	d.Set("created_at", promotion.CreatedAt.String())

	// Set the release that was actually promoted
	if promotion.Source.Release.ID != "" {
		d.Set("promoted_release_id", promotion.Source.Release.ID)
	}

	// Set configuration from API response
	d.Set("pipeline", promotion.Pipeline.ID)
	d.Set("source_app_id", promotion.Source.App.ID)

	log.Printf("[DEBUG] Pipeline promotion read completed for: %s", d.Id())
	return nil
}

func resourceHerokuPipelinePromotionDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] There is no DELETE for pipeline promotion resource so this is a no-op. Promotion will be removed from state.")
	return nil
}
