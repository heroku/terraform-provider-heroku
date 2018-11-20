package heroku

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"

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
				ValidateFunc: validateSourceUrl,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"checksum": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
							ForceNew: true,
						},

						"path": {
							Type:          schema.TypeString,
							ConflictsWith: []string{"url"},
							Optional:      true,
							ForceNew:      true,
						},

						"url": {
							Type:     schema.TypeString,
							Optional: true,
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
			if v = sourceArg["path"]; v != nil {
				return fmt.Errorf("source.checksum should be empty when source.path is set (checksum is auto-generated)")
			}
			opts.SourceBlob.Checksum = &s
		}
		if v = sourceArg["version"]; v != nil {
			s := v.(string)
			opts.SourceBlob.Version = &s
		}
		if v = sourceArg["path"]; v != nil {
			// Checksum, create, & upload source archive, setting
			// this source.url using the new source's GET URL.
			path := v.(string)
			checksum, err := checksumSource(path)
			if err != nil {
				return fmt.Errorf("Error calculating checksum for build source %s: %s", path, err)
			}
			newSource, err := client.SourceCreate(context.TODO())
			if err != nil {
				return fmt.Errorf("Error creating source for build: %s", err)
			}
			err = uploadSource(path, "PUT", newSource.SourceBlob.PutURL)
			if err != nil {
				return fmt.Errorf("Error uploading source for build to %s: %s", newSource.SourceBlob.PutURL, err)
			}
			opts.SourceBlob.URL = &newSource.SourceBlob.GetURL
			opts.SourceBlob.Checksum = &checksum
		} else if v = sourceArg["url"]; v != nil {
			s := v.(string)
			opts.SourceBlob.URL = &s
		} else {
			return fmt.Errorf("Build requires either source.path or source.url")
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

func uploadSource(filePath, httpMethod, httpUrl string) error {
	method := strings.ToUpper(httpMethod)
	log.Printf("[DEBUG] Uploading source '%s' to %s %s", filePath, method, httpUrl)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening source.path: %s", err)
	}
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Error stating source.path: %s", err)
	}
	defer file.Close()

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, httpUrl, file)
	if err != nil {
		return fmt.Errorf("Error creating source upload request: %s", err)
	}
	req.ContentLength = stat.Size()
	log.Printf("[DEBUG] Upload source request: %+v", req)
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error uploading source: %s", err)
	}

	b, err := httputil.DumpResponse(res, true)
	if err == nil {
		// generate debug output if it's available
		log.Printf("[DEBUG] Source upload response: %s", b)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Unsuccessful HTTP response from source upload: %s", res.Status)
	}

	return nil
}

func checksumSource(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("Error opening source.path: %s", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("Error reading source for checksum: %s", err)
	}
	file.Close()
	checksum := fmt.Sprintf("SHA256:%x", hash.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("Error generating source checksum: %s", err)
	}
	return checksum, nil
}

func setBuildState(d *schema.ResourceData, build *heroku.Build, appName string) error {
	d.Set("app", appName)

	var buildpacks []interface{}
	for _, buildpack := range build.Buildpacks {
		url := buildpack.URL
		buildpacks = append(buildpacks, url)
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

	if v, ok := d.GetOk("source"); ok {
		source := v.(map[string]interface{})
		// Checksum & URL are autogenerated when path is set.
		// Do not set them in that case, so state is consistent.
		if v := source["path"]; v == "" {
			if v := build.SourceBlob.Checksum; v != nil {
				source["checksum"] = *v
			}
			if v := build.SourceBlob.URL; v != "" {
				source["url"] = v
			}
		}
		if v := build.SourceBlob.Version; v != nil {
			source["version"] = *v
		}
		if err := d.Set("source", source); err != nil {
			log.Printf("[WARN] Error setting source: %s", err)
		}
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

func validateSourceUrl(v interface{}, k string) (ws []string, errors []error) {
	value := v.(map[string]interface{})["url"]
	if value == nil {
		return
	}

	pattern := `^https://`
	if !regexp.MustCompile(pattern).MatchString(value.(string)) {
		errors = append(errors, fmt.Errorf(
			"%q must be a secure URL, start with `https://`. Value is %q",
			k, value))
	}

	return
}
