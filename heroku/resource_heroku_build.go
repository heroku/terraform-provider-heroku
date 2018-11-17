package heroku

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
)

func resourceHerokuBuild() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuBuildCreate,
		Read:   resourceHerokuBuildRead,
		Delete: resourceHerokuBuildDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuBuildImport,
		},

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"buildpacks": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"output_stream_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"release_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"slug_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"source": {
				Type:         schema.TypeMap,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateSecureSourceUrl,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"checksum": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"url": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"version": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"stack": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuBuildImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, buildID := parseCompositeID(d.Id())

	build, err := client.BuildInfo(context.Background(), app, buildID)
	if err != nil {
		return nil, err
	}

	d.SetId(build.ID)
	setBuildState(d, build, app)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuBuildCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := getAppName(d)

	// Build up our creation options
	opts := heroku.BuildCreateOpts{}

	if v, ok := d.GetOk("buildpacks"); ok {
		var buildpacks []*struct {
			URL *string `json:"url,omitempty" url:"url,omitempty,key"`
		}
		buildpacksArg := v.([]interface{})
		for _, buildpack := range buildpacksArg {
			b := buildpack.(string)
			buildpacks = append(buildpacks, &struct {
				URL *string `json:"url,omitempty" url:"url,omitempty,key"`
			}{
				URL: &b,
			})
		}
		opts.Buildpacks = buildpacks
	}

	if v, ok := d.GetOk("source"); ok {
		sourceArg := v.(map[string]interface{})
		if v := sourceArg["checksum"]; v != nil {
			s := v.(string)
			opts.SourceBlob.Checksum = &s
		}
		if v = sourceArg["url"]; v != nil {
			s := v.(string)
			opts.SourceBlob.URL = &s
		}
		if v = sourceArg["version"]; v != nil {
			s := v.(string)
			opts.SourceBlob.Version = &s
		}
	}

	build, err := client.BuildCreate(context.TODO(), app, opts)
	if err != nil {
		return fmt.Errorf("Error creating build: %s opts %+v", err, opts)
	}

	d.SetId(build.ID)
	setBuildState(d, build, app)

	log.Printf("[INFO] Created build ID: %s", d.Id())
	return nil
}

func resourceHerokuBuildRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := getAppName(d)
	build, err := client.BuildInfo(context.TODO(), app, d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving build: %s", err)
	}

	setBuildState(d, build, app)

	return nil
}

// A no-op method as there is no DELETE build in Heroku Platform API.
func resourceHerokuBuildDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] There is no DELETE for build resource so this is a no-op. Build will be removed from state.")
	return nil
}

func setBuildState(d *schema.ResourceData, build *heroku.Build, appName string) error {
	d.Set("app", appName)

	var buildpacks []interface{}
	for _, buildpack := range build.Buildpacks {
		url := buildpack.URL
		buildpacks = append(buildpacks, &url)
	}
	if err := d.Set("buildpacks", buildpacks); err != nil {
		log.Printf("[WARN] Error setting buildpacks: %s", err)
	}

	d.Set("output_stream_url", build.OutputStreamURL)

	if build.Release != nil {
		d.Set("release_id", build.Release.ID)
	}

	if build.Slug != nil {
		d.Set("slug_id", build.Slug.ID)
	}

	source := map[string]interface{}{
		"checksum": &build.SourceBlob.Checksum,
		"url":      build.SourceBlob.URL,
		"version":  &build.SourceBlob.Version,
	}
	if err := d.Set("source", source); err != nil {
		log.Printf("[WARN] Error setting source: %s", err)
	}

	d.Set("stack", build.Stack)
	d.Set("status", build.Status)

	user := map[string]string{
		"email": build.User.Email,
		"id":    build.User.ID,
	}
	if err := d.Set("user", user); err != nil {
		log.Printf("[WARN] Error setting user: %s", err)
	}

	d.Set("uuid", build.ID)

	return nil
}

func validateSecureSourceUrl(v interface{}, k string) (ws []string, errors []error) {
	value := v.(map[string]interface{})["url"].(string)
	if value == "" {
		return
	}

	pattern := `^https://`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must be a secure URL, start with `https://`. Value is %q",
			k, value))
	}

	return
}
