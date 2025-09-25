package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v6"
)

func resourceHerokuTelemetryDrain() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuTelemetryDrainCreate,
		Read:   resourceHerokuTelemetryDrainRead,
		Update: resourceHerokuTelemetryDrainUpdate,
		Delete: resourceHerokuTelemetryDrainDelete,

		Schema: map[string]*schema.Schema{
			"owner_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "ID of the app or space that owns this telemetry drain",
			},

			"owner_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"app", "space"}, false),
				Description:  "Type of owner (app or space)",
			},

			"endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "URI of your OpenTelemetry consumer",
			},

			"exporter_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"otlphttp", "otlp"}, false),
				Description:  "Transport type for OpenTelemetry consumer (otlphttp or otlp)",
			},

			"signals": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "OpenTelemetry signals to send (traces, metrics, logs)",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"traces", "metrics", "logs"}, false),
				},
			},

			"headers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Headers to send to your OpenTelemetry consumer",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Computed fields
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the telemetry drain was created",
			},

			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the telemetry drain was last updated",
			},
		},
	}
}

func resourceHerokuTelemetryDrainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Validate that the owner supports OpenTelemetry drains (Fir generation only)
	ownerID := d.Get("owner_id").(string)
	ownerType := d.Get("owner_type").(string)

	if err := validateOwnerSupportsOtel(client, ownerID, ownerType); err != nil {
		return err
	}

	// Build create options
	opts := heroku.TelemetryDrainCreateOpts{
		Owner: struct {
			ID   string `json:"id" url:"id,key"`
			Type string `json:"type" url:"type,key"`
		}{
			ID:   d.Get("owner_id").(string),
			Type: d.Get("owner_type").(string),
		},
		Exporter: struct {
			Endpoint string            `json:"endpoint" url:"endpoint,key"`
			Headers  map[string]string `json:"headers,omitempty" url:"headers,omitempty,key"`
			Type     string            `json:"type" url:"type,key"`
		}{
			Endpoint: d.Get("endpoint").(string),
			Type:     d.Get("exporter_type").(string),
		},
	}

	// Convert headers
	if v, ok := d.GetOk("headers"); ok {
		opts.Exporter.Headers = convertHeaders(v.(map[string]interface{}))
	}

	// Convert signals
	opts.Signals = convertSignals(d.Get("signals").(*schema.Set))

	log.Printf("[DEBUG] Creating telemetry drain: %#v", opts)

	drain, err := client.TelemetryDrainCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("error creating telemetry drain: %s", err)
	}

	d.SetId(drain.ID)
	log.Printf("[INFO] Created telemetry drain ID: %s", drain.ID)

	return resourceHerokuTelemetryDrainRead(d, meta)
}

func resourceHerokuTelemetryDrainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	drain, err := client.TelemetryDrainInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("error retrieving telemetry drain: %s", err)
	}

	// Set computed fields
	d.Set("created_at", drain.CreatedAt.String())
	d.Set("updated_at", drain.UpdatedAt.String())

	// Set configuration from API response
	d.Set("owner_id", drain.Owner.ID)
	d.Set("owner_type", drain.Owner.Type)
	d.Set("endpoint", drain.Exporter.Endpoint)
	d.Set("exporter_type", drain.Exporter.Type)
	d.Set("headers", drain.Exporter.Headers)
	d.Set("signals", drain.Signals)

	return nil
}

func resourceHerokuTelemetryDrainUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.TelemetryDrainUpdateOpts{}

	// Build exporter update if any exporter fields changed
	if d.HasChange("endpoint") || d.HasChange("exporter_type") || d.HasChange("headers") {
		exporter := &struct {
			Endpoint string            `json:"endpoint" url:"endpoint,key"`
			Headers  map[string]string `json:"headers,omitempty" url:"headers,omitempty,key"`
			Type     string            `json:"type" url:"type,key"`
		}{
			Endpoint: d.Get("endpoint").(string),
			Type:     d.Get("exporter_type").(string),
		}

		// Convert headers
		if v, ok := d.GetOk("headers"); ok {
			exporter.Headers = convertHeaders(v.(map[string]interface{}))
		}

		opts.Exporter = exporter
	}

	// Update signals if changed
	if d.HasChange("signals") {
		opts.Signals = convertSignalsForUpdate(d.Get("signals").(*schema.Set))
	}

	log.Printf("[DEBUG] Updating telemetry drain: %#v", opts)

	_, err := client.TelemetryDrainUpdate(context.TODO(), d.Id(), opts)
	if err != nil {
		return fmt.Errorf("error updating telemetry drain: %s", err)
	}

	return resourceHerokuTelemetryDrainRead(d, meta)
}

func resourceHerokuTelemetryDrainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting telemetry drain: %s", d.Id())

	_, err := client.TelemetryDrainDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting telemetry drain: %s", err)
	}

	d.SetId("")
	return nil
}

// validateOwnerSupportsOtel checks if the owner (app or space) supports OpenTelemetry drains
func validateOwnerSupportsOtel(client *heroku.Service, ownerID, ownerType string) error {
	switch ownerType {
	case "app":
		app, err := client.AppInfo(context.TODO(), ownerID)
		if err != nil {
			return fmt.Errorf("error fetching app info: %s", err)
		}

		if !IsFeatureSupported(app.Generation.Name, "app", "otel") {
			return fmt.Errorf("telemetry drains are only supported for Fir generation apps. App '%s' is %s generation. Use heroku_drain for Cedar apps", app.Name, app.Generation.Name)
		}

	case "space":
		space, err := client.SpaceInfo(context.TODO(), ownerID)
		if err != nil {
			return fmt.Errorf("error fetching space info: %s", err)
		}

		if !IsFeatureSupported(space.Generation.Name, "space", "otel") {
			return fmt.Errorf("telemetry drains are only supported for Fir generation spaces. Space '%s' is %s generation", space.Name, space.Generation.Name)
		}

	default:
		return fmt.Errorf("invalid owner_type: %s", ownerType)
	}

	return nil
}

// convertHeaders converts map[string]interface{} to map[string]string
func convertHeaders(headers map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range headers {
		result[k] = v.(string)
	}
	return result
}

// convertSignals converts schema.Set to []string for create operations
func convertSignals(signals *schema.Set) []string {
	result := make([]string, 0, signals.Len())
	for _, signal := range signals.List() {
		result = append(result, signal.(string))
	}
	return result
}

// convertSignalsForUpdate converts schema.Set to []*string for update operations
func convertSignalsForUpdate(signals *schema.Set) []*string {
	result := make([]*string, 0, signals.Len())
	for _, signal := range signals.List() {
		s := signal.(string)
		result = append(result, &s)
	}
	return result
}
