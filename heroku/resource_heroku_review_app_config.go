package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
	"log"
	"regexp"
)

func resourceHerokuReviewAppConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHerokuReviewAppConfigCreate,
		UpdateContext: resourceHerokuReviewAppConfigUpdate,
		ReadContext:   resourceHerokuReviewAppConfigRead,
		DeleteContext: resourceHerokuReviewAppConfigDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceHerokuReviewAppConfigImport,
		},

		Schema: map[string]*schema.Schema{
			"pipeline_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"org_repo": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"deploy_target": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateDeployTargetID,
						},

						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"space", "region"}, false),
						},
					},
				},
			},

			"automatic_review_apps": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"base_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"destroy_stale_apps": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"stale_days": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				RequiredWith: []string{"destroy_stale_apps"},
				ValidateFunc: validation.IntBetween(1, 30),
			},

			"wait_for_ci": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"repo_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func validateDeployTargetID(v interface{}, k string) (ws []string, errors []error) {
	if v == nil {
		return
	}

	value := v.(string)

	pattern := `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$|^[a-z]{2}$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf("invalid deploy target id"))
	}

	return
}

func resourceHerokuReviewAppConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	var pipelineID, orgRepo string
	pipelineID, orgRepo, resultErr := parseCompositeID(d.Id())
	if resultErr != nil {
		return nil, fmt.Errorf("unable to parse import ID for pipeline ID and Github org/repo")
	}

	d.SetId(pipelineID)
	d.Set("org_repo", orgRepo)

	readErr := resourceHerokuReviewAppConfigRead(ctx, d, meta)
	if readErr.HasError() {
		return nil, fmt.Errorf("unable to import review app configs")
	}

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuReviewAppConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Config).Api
	opts := heroku.ReviewAppConfigEnableOpts{}

	pipelineID := getPipelineID(d)

	if v, ok := d.GetOk("org_repo"); ok {
		opts.Repo = v.(string)
		log.Printf("[DEBUG] review app enable - org_repo: %s", opts.Repo)
	}

	automaticReviewApps := d.Get("automatic_review_apps").(bool)
	opts.AutomaticReviewApps = &automaticReviewApps
	log.Printf("[DEBUG] review app enable - automatic_review_apps: %v", automaticReviewApps)

	destroyStaleApps := d.Get("destroy_stale_apps").(bool)
	opts.DestroyStaleApps = &destroyStaleApps
	log.Printf("[DEBUG] review app enable - destroy_stale_apps: %v", *opts.DestroyStaleApps)

	waitForCI := d.Get("wait_for_ci").(bool)
	opts.WaitForCi = &waitForCI
	log.Printf("[DEBUG] review app enable - wait_for_ci: %v", *opts.WaitForCi)

	if v, ok := d.GetOk("base_name"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] review app enable - base_name: %s", vs)
		opts.BaseName = &vs
	}

	if v, ok := d.GetOk("base_name"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] review app enable - base_name: %s", vs)
		opts.BaseName = &vs
	}

	if v, ok := d.GetOk("deploy_target"); ok {
		vL := v.([]interface{})
		deployTargetData := struct {
			ID   string
			Type string
		}{}

		for _, l := range vL {
			deployTarget := l.(map[string]interface{})

			if v, ok := deployTarget["id"]; ok && v != "" {
				vs := v.(string)
				deployTargetData.ID = vs
				log.Printf("[DEBUG] review app enable - deploy_target id: %s", deployTargetData.ID)
			}

			if v, ok := deployTarget["type"]; ok && v != "" {
				vs := v.(string)
				deployTargetData.Type = vs
				log.Printf("[DEBUG] review app enable - deploy_target type: %s", deployTargetData.Type)
			}
		}

		opts.DeployTarget = (*struct {
			ID   string `json:"id" url:"id,key"`
			Type string `json:"type" url:"type,key"`
		})(&deployTargetData)
	}

	if v, ok := d.GetOk("stale_days"); ok {
		vi := v.(int)
		log.Printf("[DEBUG] review app enable - stale_days: %d", vi)
		opts.StaleDays = &vi
	}

	log.Printf("[DEBUG] Enabling review apps config on pipeline %s", pipelineID)

	config, enableErr := client.ReviewAppConfigEnable(ctx, pipelineID, opts)
	if enableErr != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to enable review apps config for pipeline %s", pipelineID),
			Detail:   enableErr.Error(),
		})
		return diags
	}

	log.Printf("[DEBUG] Enabled review apps config on pipeline %s", pipelineID)

	// Set resource ID to the pipeline ID
	d.SetId(config.PipelineID)

	return resourceHerokuReviewAppConfigRead(ctx, d, meta)
}

func resourceHerokuReviewAppConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Config).Api
	opts := heroku.ReviewAppConfigUpdateOpts{}

	if changed := d.HasChange("automatic_review_apps"); changed {
		automaticReviewApps := d.Get("automatic_review_apps").(bool)
		opts.AutomaticReviewApps = &automaticReviewApps
		log.Printf("[DEBUG] review app update - automatic_review_apps: %v", automaticReviewApps)
	}

	if changed := d.HasChange("base_name"); changed {
		baseName := d.Get("base_name").(string)
		opts.BaseName = &baseName
		log.Printf("[DEBUG] review app update - base_name: %v", baseName)
	}

	if changed := d.HasChange("deploy_target"); changed {
		vL := d.Get("deploy_target").([]interface{})
		deployTargetData := struct {
			ID   string
			Type string
		}{}

		for _, l := range vL {
			deployTarget := l.(map[string]interface{})

			if v, ok := deployTarget["id"]; ok && v != "" {
				vs := v.(string)
				deployTargetData.ID = vs
				log.Printf("[DEBUG] review app update - deploy_target id: %s", deployTargetData.ID)
			}

			if v, ok := deployTarget["type"]; ok && v != "" {
				vs := v.(string)
				deployTargetData.Type = vs
				log.Printf("[DEBUG] review app update - deploy_target type: %s", deployTargetData.Type)
			}
		}

		opts.DeployTarget = (*struct {
			ID   string `json:"id" url:"id,key"`
			Type string `json:"type" url:"type,key"`
		})(&deployTargetData)
	}

	if changed := d.HasChange("destroy_stale_apps"); changed {
		destroyStaleApps := d.Get("destroy_stale_apps").(bool)
		opts.DestroyStaleApps = &destroyStaleApps
		log.Printf("[DEBUG] review app update - destroy_stale_apps: %v", destroyStaleApps)
	}

	if changed := d.HasChange("stale_days"); changed {
		staleDays := d.Get("stale_days").(int)
		opts.StaleDays = &staleDays
		log.Printf("[DEBUG] review app update - stale_days: %v", staleDays)
	}

	if changed := d.HasChange("wait_for_ci"); changed {
		waitForCI := d.Get("wait_for_ci").(bool)
		opts.WaitForCi = &waitForCI
		log.Printf("[DEBUG] review app update - wait_for_ci: %v", waitForCI)
	}

	log.Printf("[DEBUG] Updating review apps config on pipeline %s", d.Id())

	_, updateErr := client.ReviewAppConfigUpdate(ctx, d.Id(), opts)
	if updateErr != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to update review apps config for pipeline %s", d.Id()),
			Detail:   updateErr.Error(),
		})
		return diags
	}

	log.Printf("[DEBUG] Updated review apps config on pipeline %s", d.Id())

	return resourceHerokuReviewAppConfigRead(ctx, d, meta)
}

func resourceHerokuReviewAppConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Config).Api

	reviewAppConfig, readErr := client.ReviewAppConfigInfo(ctx, d.Id())
	if readErr != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to retrieve review apps config for pipeline %s", d.Id()),
			Detail:   readErr.Error(),
		})
		return diags
	}

	d.Set("pipeline_id", reviewAppConfig.PipelineID)
	d.Set("automatic_review_apps", reviewAppConfig.AutomaticReviewApps)
	d.Set("base_name", reviewAppConfig.BaseName)
	d.Set("destroy_stale_apps", reviewAppConfig.DestroyStaleApps)
	d.Set("stale_days", reviewAppConfig.StaleDays)
	d.Set("wait_for_ci", reviewAppConfig.WaitForCi)
	d.Set("repo_id", reviewAppConfig.Repo.ID)

	deployTarget := make([]map[string]interface{}, 0)
	if reviewAppConfig.DeployTarget != nil {
		// Lookup region info as the /review-app-config endpoint returns the region UUID
		// for the deploy target ID instead of the name (ex. 'us').
		region, regionGetErr := client.RegionInfo(ctx, reviewAppConfig.DeployTarget.ID)
		if regionGetErr != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Unable to retrieve region %s", reviewAppConfig.DeployTarget.ID),
				Detail:   regionGetErr.Error(),
			})
			return diags
		}

		deployTarget = append(deployTarget, map[string]interface{}{
			"id":   region.Name,
			"type": reviewAppConfig.DeployTarget.Type,
		})
	}
	d.Set("deploy_target", deployTarget)

	return diags
}

func resourceHerokuReviewAppConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Config).Api

	log.Printf("[DEBUG] Disabling review apps config on pipeline %s", d.Id())

	_, disableErr := client.ReviewAppConfigDelete(ctx, d.Id())
	if disableErr != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to disable review apps config for pipeline %s", d.Id()),
			Detail:   disableErr.Error(),
		})
		return diags
	}

	d.SetId("")

	log.Printf("[DEBUG] Disabled review apps config on pipeline %s", d.Id())

	return diags
}
