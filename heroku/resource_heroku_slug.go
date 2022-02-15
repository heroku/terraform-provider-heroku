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

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuSlug() *schema.Resource {
	return &schema.Resource{
		Create:        resourceHerokuSlugCreate,
		Read:          resourceHerokuSlugRead,
		Delete:        resourceHerokuSlugDelete,
		CustomizeDiff: resourceHerokuSlugCustomizeDiff,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuSlugImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Local tarball to be uploaded after slug creation
			"file_path": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// https:// URL of tarball to upload into slug
			"file_url": {
				Type:          schema.TypeString,
				ConflictsWith: []string{"file_path"},
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validateFileUrl,
			},

			"blob": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"method": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"url": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"buildpack_provided_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"checksum": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"commit": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"commit_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"process_types": {
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: true,
			},

			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// Create/argument: either a name or UUID.
			// Read/attribute: name of the stack.
			"stack": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"stack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuSlugV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeAppToAppID,
				Version: 0,
			},
		},
	}
}

func resourceHerokuSlugImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, slugID, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	slug, err := client.SlugInfo(context.Background(), app, slugID)
	if err != nil {
		return nil, err
	}

	d.SetId(slug.ID)

	foundApp, err := resourceHerokuAppRetrieve(app, client)
	if err != nil {
		return nil, err
	}

	d.Set("app_id", foundApp.App.ID)

	setErr := setSlugState(d, slug)
	if setErr != nil {
		return nil, setErr
	}

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuSlugCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appID := getAppId(d)

	// Build up our creation options
	opts := heroku.SlugCreateOpts{}

	if pt, ok := d.GetOk("process_types"); ok {
		opts.ProcessTypes = make(map[string]string)

		for k, v := range pt.(map[string]interface{}) {
			opts.ProcessTypes[k] = v.(string)
		}
	}

	if v, ok := d.GetOk("buildpack_provided_description"); ok {
		opts.BuildpackProvidedDescription = heroku.String(v.(string))
	}

	// Only file_path or file_url will be set, because of ConflictsWith.
	var filePath string
	// Simply use the configured file path
	if v, ok := d.GetOk("file_path"); ok {
		filePath = v.(string)
	}
	// Download the slug archive to a unique file path and clean it up
	// after uploading to Heroku platform.
	if v, ok := d.GetOk("file_url"); ok {
		fileUrl := v.(string)

		newUuid, err := uuid.GenerateUUID()
		if err != nil {
			return err
		}
		filePath = fmt.Sprintf("slug-%s.tgz", newUuid)

		err = downloadSlug(fileUrl, filePath)
		if err != nil {
			return err
		}
		defer cleanupSlugFile(filePath)
	}

	// Require a file path by validating this programmatically.
	// (ConflictsWith cannot be used with Required)
	if filePath == "" {
		return fmt.Errorf("Error creating slug: requires either `file_path` or `file_url` attribute")
	}

	if v, ok := d.GetOk("checksum"); ok {
		// Use specified checksum when its set
		opts.Checksum = heroku.String(v.(string))
	} else {
		// Optionally capture the checksum (really sha256 hash) of the slug file.
		if filePath != "" {
			checksum, err := checksumSlug(filePath)
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] Slug checksum: %s", checksum)
			opts.Checksum = heroku.String(checksum)
		}
	}

	if v, ok := d.GetOk("commit"); ok {
		opts.Commit = heroku.String(v.(string))
	}
	if v, ok := d.GetOk("commit_description"); ok {
		opts.CommitDescription = heroku.String(v.(string))
	}
	if v, ok := d.GetOk("stack"); ok {
		opts.Stack = heroku.String(v.(string))
	}

	slug, err := client.SlugCreate(context.TODO(), appID, opts)
	if err != nil {
		return fmt.Errorf("Error creating slug: %s opts %+v", err, opts)
	}

	// Optionally upload slug before setting ID, so that an upload failure
	// causes a resource creation error, is not saved in state.
	if filePath != "" {
		err := uploadSlug(filePath, slug.Blob.Method, slug.Blob.URL)
		if err != nil {
			return err
		}
	}

	d.SetId(slug.ID)

	setErr := setSlugState(d, slug)
	if setErr != nil {
		return setErr
	}

	log.Printf("[INFO] Created slug ID: %s", d.Id())
	return nil
}

func resourceHerokuSlugRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appID := getAppId(d)
	slug, err := client.SlugInfo(context.TODO(), appID, d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving slug: %s", err)
	}

	setErr := setSlugState(d, slug)
	if setErr != nil {
		return setErr
	}

	return nil
}

// A no-op method as there is no DELETE slug in Heroku Platform API.
func resourceHerokuSlugDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] There is no DELETE for slug resource so this is a no-op. Slug will be removed from state.")
	return nil
}

func resourceHerokuSlugCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
	// Detect changes to the content of local slug archive.
	if v, ok := diff.GetOk("file_path"); ok {
		filePath := v.(string)
		realChecksum, err := checksumSlug(filePath)
		if err == nil {
			oldChecksum, newChecksum := diff.GetChange("checksum")
			log.Printf("[DEBUG] Diffing slug: old '%s', new '%s', real '%s'", oldChecksum, newChecksum, realChecksum)
			if newChecksum != realChecksum {
				if err := diff.SetNew("checksum", realChecksum); err != nil {
					return fmt.Errorf("Error updating slug archive checksum: %s", err)
				}
				if err := diff.ForceNew("checksum"); err != nil {
					return fmt.Errorf("Error forcing new slug resource: %s", err)
				}
			}
		}
	}

	return nil
}

func downloadSlug(httpUrl, destinationFilePath string) error {
	log.Printf("[DEBUG] Downloading slug from %s", httpUrl)

	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", httpUrl, nil)
	if err != nil {
		return fmt.Errorf("Error creating slug download request: %s (%s)", err, httpUrl)
	}
	log.Printf("[DEBUG] Download slug request: %+v", req)
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error downloading slug: %s (%s)", err, httpUrl)
	}

	b, err := httputil.DumpResponse(res, true)
	if err == nil {
		// generate debug output if it's available
		log.Printf("[DEBUG] Slug download response: %s", b)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Unsuccessful HTTP response from slug download: %s (%s)", res.Status, httpUrl)
	}

	slugFile, err := os.Create(destinationFilePath)
	if err != nil {
		return fmt.Errorf("Error creating slug file: %s (%s)", err, destinationFilePath)
	}
	defer slugFile.Close()

	_, copyErr := io.Copy(slugFile, res.Body)
	if copyErr != nil {
		return copyErr
	}

	return nil
}

func uploadSlug(filePath, httpMethod, httpUrl string) error {
	method := strings.ToUpper(httpMethod)
	log.Printf("[DEBUG] Uploading slug '%s' to %s %s", filePath, method, httpUrl)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening slug file_path: %s", err)
	}
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Error stating slug file_path: %s", err)
	}
	defer file.Close()

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, httpUrl, file)
	if err != nil {
		return fmt.Errorf("Error creating slug upload request: %s", err)
	}
	req.ContentLength = stat.Size()
	log.Printf("[DEBUG] Upload slug request: %+v", req)
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error uploading slug: %s", err)
	}

	b, err := httputil.DumpResponse(res, true)
	if err == nil {
		// generate debug output if it's available
		log.Printf("[DEBUG] Slug upload response: %s", b)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Unsuccessful HTTP response from slug upload: %s", res.Status)
	}
	return nil
}

func checksumSlug(filePath string) (string, error) {
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return "", fmt.Errorf("Error opening slug file_path: %s", openErr)
	}

	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", fmt.Errorf("Error reading slug for checksum: %s", copyErr)
	}

	closeErr := file.Close()
	if closeErr != nil {
		return "", closeErr
	}

	checksum := fmt.Sprintf("SHA256:%x", hash.Sum(nil))
	return checksum, nil
}

func setSlugState(d *schema.ResourceData, slug *heroku.Slug) error {
	blob := []map[string]string{{
		"method": slug.Blob.Method,
		"url":    slug.Blob.URL,
	}}
	if err := d.Set("blob", blob); err != nil {
		log.Printf("[WARN] Error setting blob: %s", err)
	}
	d.Set("buildpack_provided_description", slug.BuildpackProvidedDescription)
	d.Set("checksum", slug.Checksum)
	d.Set("commit", slug.Commit)
	d.Set("commit_description", slug.CommitDescription)
	if err := d.Set("process_types", slug.ProcessTypes); err != nil {
		log.Printf("[WARN] Error setting process_types: %s", err)
	}
	d.Set("size", slug.Size)
	d.Set("stack_id", slug.Stack.ID)
	d.Set("stack", slug.Stack.Name)
	return nil
}

func cleanupSlugFile(filePath string) {
	if filePath != "" {
		err := os.Remove(filePath)
		if err != nil {
			log.Printf("[WARN] Error cleaning-up downloaded slug: %s (%s)", err, filePath)
		}
	}
}

func validateFileUrl(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
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

func resourceHerokuSlugV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// Local tarball to be uploaded after slug creation
			"file_path": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// https:// URL of tarball to upload into slug
			"file_url": {
				Type:          schema.TypeString,
				ConflictsWith: []string{"file_path"},
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validateFileUrl,
			},

			"blob": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"method": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"url": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"buildpack_provided_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"checksum": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"commit": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"commit_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"process_types": {
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: true,
			},

			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// Create/argument: either a name or UUID.
			// Read/attribute: name of the stack.
			"stack": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"stack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
