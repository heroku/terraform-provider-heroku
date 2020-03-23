package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
	"log"
)

func resourceHerokuPipelineConfigVar() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuPipelineConfigVarCreate,
		Update: resourceHerokuPipelineConfigVarUpdate,
		Read:   resourceHerokuPipelineConfigVarRead,
		Delete: resourceHerokuPipelineConfigVarDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuPipelineConfigVarImport,
		},

		Schema: map[string]*schema.Schema{
			"pipeline_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateUUID,
			},

			"pipeline_stage": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"test", "review"}, false),
			},

			"vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"sensitive_vars": {
				Type:      schema.TypeMap,
				Sensitive: true,
				Optional:  true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
			},

			"all_vars": {
				Type:     schema.TypeMap,
				Computed: true,
				// These are marked Sensitive so that "sensitive_config_vars" do not
				// leak in the console/logs.
				Sensitive: true,
			},
		},
	}
}

func resourceHerokuPipelineConfigVarImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	noImportErr := fmt.Errorf("not possible to import this resource")

	return nil, noImportErr
}

func resourceHerokuPipelineConfigVarCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	pipelineID := getPipelineID(d)
	pipelineStage := getPipelineStage(d)
	vars := getVars(d)
	sensitiveVars := getSensitiveVars(d)

	// Check for duplicates between vars & sensitiveVars
	dupeErr := duplicateVarsChecker(vars, sensitiveVars)
	if dupeErr != nil {
		return dupeErr
	}

	// Combine both sensitive and non-sensitive vars
	combinedVars := mergeVars(vars, sensitiveVars)

	log.Printf("[INFO] Creating pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	// Update the vars for the pipeline
	updateErr := updatePipelineConfigVars(client, pipelineID, pipelineStage, nil, combinedVars)
	if updateErr != nil {
		return updateErr
	}

	log.Printf("[INFO] Created pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	// Set the ID to be pipeline ID + stage
	d.SetId(fmt.Sprintf("%s:%s", pipelineID, pipelineStage))

	return resourceHerokuPipelineConfigVarRead(d, meta)
}

func resourceHerokuPipelineConfigVarUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	pipelineID := getPipelineID(d)
	pipelineStage := getPipelineStage(d)

	var oldVars, newVars, oldSensitiveVars, newSensitiveVars, allOldVars, allNewVars map[string]interface{}
	oldVars, newVars = getVarDiff(d, "vars")
	oldSensitiveVars, newSensitiveVars = getVarDiff(d, "sensitive_vars")

	// Merge the vars
	allOldVars = mergeVars(oldVars, oldSensitiveVars)
	allNewVars = mergeVars(newVars, newSensitiveVars)

	log.Printf("[INFO] Updating pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	// Update the vars for the pipeline
	updateErr := updatePipelineConfigVars(client, pipelineID, pipelineStage, allOldVars, allNewVars)
	if updateErr != nil {
		return updateErr
	}

	log.Printf("[INFO] Updated pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	return resourceHerokuPipelineConfigVarRead(d, meta)
}

func resourceHerokuPipelineConfigVarRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Parse the resource ID to return the pipeline ID & stage
	pipelineID, pipelineStage, parseErr := parseCompositeID(d.Id())
	if parseErr != nil {
		return parseErr
	}

	remotePipelineVars, getErr := client.PipelineConfigVarInfoForApp(context.TODO(), pipelineID, pipelineStage)
	if getErr != nil {
		return getErr
	}

	// Need to convert remotePipelineVars to a data type required by vetVarsForState
	rpvFormatted := make(map[string]string)
	for key, value := range remotePipelineVars {
		rpvFormatted[key] = *value
	}

	vettedConfigVars, vettedSensitiveConfigVars := vetVarsForState(getVars(d), getSensitiveVars(d), rpvFormatted)

	log.Printf("[DEBUG] pipeline config vars to be set in state: *%#v", vettedConfigVars)
	log.Printf("[DEBUG] pipeline sensitive config vars to be set in state: *%#v", vettedSensitiveConfigVars)

	var setErr error
	setErr = d.Set("pipeline_id", pipelineID)
	setErr = d.Set("pipeline_stage", pipelineStage)
	setErr = d.Set("vars", vettedConfigVars)
	setErr = d.Set("sensitive_vars", vettedSensitiveConfigVars)
	setErr = d.Set("all_vars", rpvFormatted)

	return setErr
}

func resourceHerokuPipelineConfigVarDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	pipelineID := getPipelineID(d)
	pipelineStage := getPipelineStage(d)

	vars := getVars(d)
	sensitiveVars := getSensitiveVars(d)
	allVars := mergeVars(vars, sensitiveVars)

	log.Printf("[INFO] Removing pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	// Delete all config vars by setting the vars defined in resource schema to nil value.
	updateErr := updatePipelineConfigVars(client, pipelineID, pipelineStage, allVars, nil)
	if updateErr != nil {
		return updateErr
	}

	log.Printf("[INFO] Removed pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	return nil
}

func updatePipelineConfigVars(client *heroku.Service, pipelineID, pipelineStage string,
	oldVars, newVars map[string]interface{}) error {
	varsToModify := constructVars(oldVars, newVars)

	log.Printf("[INFO] Modifying pipeline [%s] stage [%s] config vars: *%#v", pipelineID, pipelineStage, varsToModify)

	if _, updateErr := client.PipelineConfigVarUpdate(context.TODO(), pipelineID, pipelineStage, varsToModify); updateErr != nil {
		return fmt.Errorf("error updating pipeline config vars: %s", updateErr)
	}

	log.Printf("[INFO] Modifying pipeline [%s] stage [%s] config vars: *%#v", pipelineID, pipelineStage, varsToModify)

	return nil
}

func getPipelineID(d *schema.ResourceData) string {
	var pipelineID string
	if v, ok := d.GetOk("pipeline_id"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] pipeline ID: %s", vs)
		pipelineID = vs
	}

	return pipelineID
}

func getPipelineStage(d *schema.ResourceData) string {
	var stage string
	if v, ok := d.GetOk("pipeline_stage"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] pipeline stage: %s", vs)
		stage = vs
	}

	return stage
}
