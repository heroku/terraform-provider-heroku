package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuPipelineCoupling() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuPipelineCouplingCreate,
		Read:   resourceHerokuPipelineCouplingRead,
		Delete: resourceHerokuPipelineCouplingDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"pipeline": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"stage": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice(
					[]string{"review", "development", "staging", "production"},
					false,
				),
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuPipelineCouplingV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeAppToAppID,
				Version: 0,
			},
		},
	}
}

func resourceHerokuPipelineCouplingCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.PipelineCouplingCreateOpts{
		App:      d.Get("app_id").(string),
		Pipeline: d.Get("pipeline").(string),
		Stage:    d.Get("stage").(string),
	}

	log.Printf("[DEBUG] PipelineCoupling create configuration: %#v", opts)

	p, err := client.PipelineCouplingCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("Error creating pipeline: %s", err)
	}

	d.SetId(p.ID)

	log.Printf("[INFO] PipelineCoupling ID: %s", d.Id())

	return resourceHerokuPipelineCouplingRead(d, meta)
}

func resourceHerokuPipelineCouplingDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting pipeline: %s", d.Id())

	_, err := client.PipelineCouplingDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting pipeline: %s", err)
	}

	return nil
}

func resourceHerokuPipelineCouplingRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	p, err := client.PipelineCouplingInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving pipeline: %s", err)
	}

	d.Set("app_id", p.App.ID)
	d.Set("stage", p.Stage)
	d.Set("pipeline", p.Pipeline.ID)

	return nil
}

func resourceHerokuPipelineCouplingV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"pipeline": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"stage": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice(
					[]string{"review", "development", "staging", "production"},
					false,
				),
			},
		},
	}
}
