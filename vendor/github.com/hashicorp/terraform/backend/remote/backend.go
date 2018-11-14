package remote

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
)

const (
	defaultHostname    = "app.terraform.io"
	defaultParallelism = 10
	serviceID          = "tfe.v2"
)

// Remote is an implementation of EnhancedBackend that performs all
// operations in a remote backend.
type Remote struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// ShowDiagnostics prints diagnostic messages to the UI.
	ShowDiagnostics func(vals ...interface{})

	// ContextOpts are the base context options to set when initializing a
	// new Terraform context. Many of these will be overridden or merged by
	// Operation. See Operation for more details.
	ContextOpts *terraform.ContextOpts

	// client is the remote backend API client
	client *tfe.Client

	// hostname of the remote backend server
	hostname string

	// organization is the organization that contains the target workspaces
	organization string

	// workspace is used to map the default workspace to a remote workspace
	workspace string

	// prefix is used to filter down a set of workspaces that use a single
	// configuration
	prefix string

	// schema defines the configuration for the backend
	schema *schema.Backend

	// services is used for service discovery
	services *disco.Disco

	// opLock locks operations
	opLock sync.Mutex
}

var _ backend.Backend = (*Remote)(nil)

// New creates a new initialized remote backend.
func New(services *disco.Disco) *Remote {
	return &Remote{
		services: services,
	}
}

func (b *Remote) ConfigSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"hostname": {
				Type:        cty.String,
				Optional:    true,
				Description: schemaDescriptions["hostname"],
			},
			"organization": {
				Type:        cty.String,
				Required:    true,
				Description: schemaDescriptions["organization"],
			},
			"token": {
				Type:        cty.String,
				Optional:    true,
				Description: schemaDescriptions["token"],
			},
		},

		BlockTypes: map[string]*configschema.NestedBlock{
			"workspaces": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"name": {
							Type:        cty.String,
							Optional:    true,
							Description: schemaDescriptions["name"],
						},
						"prefix": {
							Type:        cty.String,
							Optional:    true,
							Description: schemaDescriptions["prefix"],
						},
					},
				},
				Nesting: configschema.NestingSingle,
			},
		},
	}
}

func (b *Remote) ValidateConfig(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if val := obj.GetAttr("organization"); !val.IsNull() {
		if val.AsString() == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid organization value",
				`The "organization" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "organization"}},
			))
		}
	}

	var name, prefix string
	if workspaces := obj.GetAttr("workspaces"); !workspaces.IsNull() {
		if val := workspaces.GetAttr("name"); !val.IsNull() {
			name = val.AsString()
		}
		if val := workspaces.GetAttr("prefix"); !val.IsNull() {
			prefix = val.AsString()
		}
	}

	// Make sure that we have either a workspace name or a prefix.
	if name == "" && prefix == "" {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid workspaces configuration",
			`Either workspace "name" or "prefix" is required.`,
			cty.Path{cty.GetAttrStep{Name: "workspaces"}},
		))
	}

	// Make sure that only one of workspace name or a prefix is configured.
	if name != "" && prefix != "" {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid workspaces configuration",
			`Only one of workspace "name" or "prefix" is allowed.`,
			cty.Path{cty.GetAttrStep{Name: "workspaces"}},
		))
	}

	return diags
}

func (b *Remote) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Get the hostname.
	if val := obj.GetAttr("hostname"); !val.IsNull() && val.AsString() != "" {
		b.hostname = val.AsString()
	} else {
		b.hostname = defaultHostname
	}

	// Get the organization.
	if val := obj.GetAttr("organization"); !val.IsNull() {
		b.organization = val.AsString()
	}

	// Get the workspaces configuration block and retrieve the
	// default workspace name and prefix.
	if workspaces := obj.GetAttr("workspaces"); !workspaces.IsNull() {
		if val := workspaces.GetAttr("name"); !val.IsNull() {
			b.workspace = val.AsString()
		}
		if val := workspaces.GetAttr("prefix"); !val.IsNull() {
			b.prefix = val.AsString()
		}
	}

	// Discover the service URL for this host to confirm that it provides
	// a remote backend API and to discover the required base path.
	service, err := b.discover(b.hostname)
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			strings.ToUpper(err.Error()[:1])+err.Error()[1:],
			`If you are sure the hostname is correct, this could also indicate SSL `+
				`verification issues. Please use "openssl s_client -connect <HOST>" to `+
				`identify any certificate or certificate chain issues.`,
			cty.Path{cty.GetAttrStep{Name: "hostname"}},
		))
		return diags
	}

	// Retrieve the token for this host as configured in the credentials
	// section of the CLI Config File.
	token, err := b.token(b.hostname)
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			strings.ToUpper(err.Error()[:1])+err.Error()[1:],
			`If you are sure the hostname is correct, this could also indicate SSL `+
				`verification issues. Please use "openssl s_client -connect <HOST>" to `+
				`identify any certificate or certificate chain issues.`,
			cty.Path{cty.GetAttrStep{Name: "hostname"}},
		))
		return diags
	}
	if token == "" {
		if val := obj.GetAttr("token"); !val.IsNull() {
			token = val.AsString()
		}
	}

	cfg := &tfe.Config{
		Address:  service.String(),
		BasePath: service.Path,
		Token:    token,
		Headers:  make(http.Header),
	}

	// Set the version header to the current version.
	cfg.Headers.Set(version.Header, version.Version)

	// Create the remote backend API client.
	b.client, err = tfe.NewClient(cfg)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create the Terraform Enterprise client",
			fmt.Sprintf(
				`The "remote" backend encountered an unexpected error while creating the `+
					`Terraform Enterprise client: %s.`, err,
			),
		))
	}

	return diags
}

// discover the remote backend API service URL and token.
func (b *Remote) discover(hostname string) (*url.URL, error) {
	host, err := svchost.ForComparison(hostname)
	if err != nil {
		return nil, err
	}
	service := b.services.DiscoverServiceURL(host, serviceID)
	if service == nil {
		return nil, fmt.Errorf("host %s does not provide a remote backend API", host)
	}
	return service, nil
}

// token returns the token for this host as configured in the credentials
// section of the CLI Config File. If no token was configured, an empty
// string will be returned instead.
func (b *Remote) token(hostname string) (string, error) {
	host, err := svchost.ForComparison(hostname)
	if err != nil {
		return "", err
	}
	creds, err := b.services.CredentialsForHost(host)
	if err != nil {
		log.Printf("[WARN] Failed to get credentials for %s: %s (ignoring)", host, err)
		return "", nil
	}
	if creds != nil {
		return creds.Token(), nil
	}
	return "", nil
}

// Workspaces returns a filtered list of remote workspace names.
func (b *Remote) Workspaces() ([]string, error) {
	if b.prefix == "" {
		return nil, backend.ErrWorkspacesNotSupported
	}
	return b.workspaces()
}

func (b *Remote) workspaces() ([]string, error) {
	// Check if the configured organization exists.
	_, err := b.client.Organizations.Read(context.Background(), b.organization)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			return nil, fmt.Errorf("organization %s does not exist", b.organization)
		}
		return nil, err
	}

	options := tfe.WorkspaceListOptions{}
	switch {
	case b.workspace != "":
		options.Search = tfe.String(b.workspace)
	case b.prefix != "":
		options.Search = tfe.String(b.prefix)
	}

	// Create a slice to contain all the names.
	var names []string

	for {
		wl, err := b.client.Workspaces.List(context.Background(), b.organization, options)
		if err != nil {
			return nil, err
		}

		for _, w := range wl.Items {
			if b.workspace != "" && w.Name == b.workspace {
				names = append(names, backend.DefaultStateName)
				continue
			}
			if b.prefix != "" && strings.HasPrefix(w.Name, b.prefix) {
				names = append(names, strings.TrimPrefix(w.Name, b.prefix))
			}
		}

		// Exit the loop when we've seen all pages.
		if wl.CurrentPage >= wl.TotalPages {
			break
		}

		// Update the page number to get the next page.
		options.PageNumber = wl.NextPage
	}

	// Sort the result so we have consistent output.
	sort.StringSlice(names).Sort()

	return names, nil
}

// DeleteWorkspace removes the remote workspace if it exists.
func (b *Remote) DeleteWorkspace(name string) error {
	if b.workspace == "" && name == backend.DefaultStateName {
		return backend.ErrDefaultWorkspaceNotSupported
	}
	if b.prefix == "" && name != backend.DefaultStateName {
		return backend.ErrWorkspacesNotSupported
	}

	// Configure the remote workspace name.
	switch {
	case name == backend.DefaultStateName:
		name = b.workspace
	case b.prefix != "" && !strings.HasPrefix(name, b.prefix):
		name = b.prefix + name
	}

	// Check if the configured organization exists.
	_, err := b.client.Organizations.Read(context.Background(), b.organization)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			return fmt.Errorf("organization %s does not exist", b.organization)
		}
		return err
	}

	client := &remoteClient{
		client:       b.client,
		organization: b.organization,
		workspace:    name,
	}

	return client.Delete()
}

// StateMgr returns the latest state of the given remote workspace. The
// workspace will be created if it doesn't exist.
func (b *Remote) StateMgr(name string) (state.State, error) {
	if b.workspace == "" && name == backend.DefaultStateName {
		return nil, backend.ErrDefaultWorkspaceNotSupported
	}
	if b.prefix == "" && name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}

	workspaces, err := b.workspaces()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving workspaces: %v", err)
	}

	exists := false
	for _, workspace := range workspaces {
		if name == workspace {
			exists = true
			break
		}
	}

	// Configure the remote workspace name.
	switch {
	case name == backend.DefaultStateName:
		name = b.workspace
	case b.prefix != "" && !strings.HasPrefix(name, b.prefix):
		name = b.prefix + name
	}

	if !exists {
		options := tfe.WorkspaceCreateOptions{
			Name: tfe.String(name),
		}

		// We only set the Terraform Version for the new workspace if this is
		// a release candidate or a final release.
		if version.Prerelease == "" || strings.HasPrefix(version.Prerelease, "rc") {
			options.TerraformVersion = tfe.String(version.String())
		}

		_, err = b.client.Workspaces.Create(context.Background(), b.organization, options)
		if err != nil {
			return nil, fmt.Errorf("Error creating workspace %s: %v", name, err)
		}
	}

	client := &remoteClient{
		client:       b.client,
		organization: b.organization,
		workspace:    name,

		// This is optionally set during Terraform Enterprise runs.
		runID: os.Getenv("TFE_RUN_ID"),
	}

	return &remote.State{Client: client}, nil
}

// Operation implements backend.Enhanced
func (b *Remote) Operation(ctx context.Context, op *backend.Operation) (*backend.RunningOperation, error) {
	// Configure the remote workspace name.
	switch {
	case op.Workspace == backend.DefaultStateName:
		op.Workspace = b.workspace
	case b.prefix != "" && !strings.HasPrefix(op.Workspace, b.prefix):
		op.Workspace = b.prefix + op.Workspace
	}

	// Determine the function to call for our operation
	var f func(context.Context, context.Context, *backend.Operation) (*tfe.Run, error)
	switch op.Type {
	case backend.OperationTypePlan:
		f = b.opPlan
	case backend.OperationTypeApply:
		f = b.opApply
	default:
		return nil, fmt.Errorf(
			"\n\nThe \"remote\" backend does not support the %q operation.\n"+
				"Please use the remote backend web UI for running this operation:\n"+
				"https://%s/app/%s/%s", op.Type, b.hostname, b.organization, op.Workspace)
	}

	// Lock
	b.opLock.Lock()

	// Build our running operation
	// the runninCtx is only used to block until the operation returns.
	runningCtx, done := context.WithCancel(context.Background())
	runningOp := &backend.RunningOperation{
		Context:   runningCtx,
		PlanEmpty: true,
	}

	// stopCtx wraps the context passed in, and is used to signal a graceful Stop.
	stopCtx, stop := context.WithCancel(ctx)
	runningOp.Stop = stop

	// cancelCtx is used to cancel the operation immediately, usually
	// indicating that the process is exiting.
	cancelCtx, cancel := context.WithCancel(context.Background())
	runningOp.Cancel = cancel

	// Do it.
	go func() {
		defer done()
		defer stop()
		defer cancel()

		defer b.opLock.Unlock()

		r, opErr := f(stopCtx, cancelCtx, op)
		if opErr != nil && opErr != context.Canceled {
			b.ReportResult(runningOp, opErr)
			return
		}

		if r != nil {
			// Retrieve the run to get its current status.
			r, err := b.client.Runs.Read(cancelCtx, r.ID)
			if err != nil {
				b.ReportResult(runningOp, generalError("Failed to retrieve run", err))
				return
			}

			// Record if there are any changes.
			runningOp.PlanEmpty = !r.HasChanges

			if opErr == context.Canceled {
				if err := b.cancel(cancelCtx, op, r); err != nil {
					b.ReportResult(runningOp, generalError("Failed to retrieve run", err))
					return
				}
			}

			if r.Status == tfe.RunErrored {
				runningOp.Result = backend.OperationFailure
			}
		}
	}()

	// Return the running operation.
	return runningOp, nil
}

func (b *Remote) cancel(cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	if r.Status == tfe.RunPending && r.Actions.IsCancelable {
		// Only ask if the remote operation should be canceled
		// if the auto approve flag is not set.
		if !op.AutoApprove {
			v, err := op.UIIn.Input(&terraform.InputOpts{
				Id:          "cancel",
				Query:       "\nDo you want to cancel the pending remote operation?",
				Description: "Only 'yes' will be accepted to cancel.",
			})
			if err != nil {
				return generalError("Failed asking to cancel", err)
			}
			if v != "yes" {
				if b.CLI != nil {
					b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationNotCanceled)))
				}
				return nil
			}
		} else {
			if b.CLI != nil {
				// Insert a blank line to separate the ouputs.
				b.CLI.Output("")
			}
		}

		// Try to cancel the remote operation.
		err := b.client.Runs.Cancel(cancelCtx, r.ID, tfe.RunCancelOptions{})
		if err != nil {
			return generalError("Failed to cancel run", err)
		}
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationCanceled)))
		}
	}

	return nil
}

// ReportResult is a helper for the common chore of setting the status of
// a running operation and showing any diagnostics produced during that
// operation.
//
// If the given diagnostics contains errors then the operation's result
// will be set to backend.OperationFailure. It will be set to
// backend.OperationSuccess otherwise. It will then use b.ShowDiagnostics
// to show the given diagnostics before returning.
//
// Callers should feel free to do each of these operations separately in
// more complex cases where e.g. diagnostics are interleaved with other
// output, but terminating immediately after reporting error diagnostics is
// common and can be expressed concisely via this method.
func (b *Remote) ReportResult(op *backend.RunningOperation, err error) {
	var diags tfdiags.Diagnostics

	diags = diags.Append(err)
	if diags.HasErrors() {
		op.Result = backend.OperationFailure
	} else {
		op.Result = backend.OperationSuccess
	}

	if b.ShowDiagnostics != nil {
		b.ShowDiagnostics(diags)
	} else {
		// Shouldn't generally happen, but if it does then we'll at least
		// make some noise in the logs to help us spot it.
		if len(diags) != 0 {
			log.Printf(
				"[ERROR] Remote backend needs to report diagnostics but ShowDiagnostics is not set:\n%s",
				diags.ErrWithWarnings(),
			)
		}
	}
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is guaranteed to always return a non-nil value and so useful
// as a helper to wrap any potentially colored strings.
// func (b *Remote) Colorize() *colorstring.Colorize {
// 	if b.CLIColor != nil {
// 		return b.CLIColor
// 	}

// 	return &colorstring.Colorize{
// 		Colors:  colorstring.DefaultColors,
// 		Disable: true,
// 	}
// }

func generalError(msg string, err error) error {
	var diags tfdiags.Diagnostics

	if urlErr, ok := err.(*url.Error); ok {
		err = urlErr.Err
	}

	switch err {
	case context.Canceled:
		return err
	case tfe.ErrResourceNotFound:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("%s: %v", msg, err),
			`The configured "remote" backend returns '404 Not Found' errors for resources `+
				`that do not exist, as well as for resources that a user doesn't have access `+
				`to. When the resource does exists, please check the rights for the used token.`,
		))
		return diags.Err()
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("%s: %v", msg, err),
			`The configured "remote" backend encountered an unexpected error. Sometimes `+
				`this is caused by network connection problems, in which case you could retry `+
				`the command. If the issue persists please open a support ticket to get help `+
				`resolving the problem.`,
		))
		return diags.Err()
	}
}

const operationCanceled = `
[reset][red]The remote operation was successfully cancelled.[reset]
`

const operationNotCanceled = `
[reset][red]The remote operation was not cancelled.[reset]
`

var schemaDescriptions = map[string]string{
	"hostname":     "The remote backend hostname to connect to (defaults to app.terraform.io).",
	"organization": "The name of the organization containing the targeted workspace(s).",
	"token": "The token used to authenticate with the remote backend. If credentials for the\n" +
		"host are configured in the CLI Config File, then those will be used instead.",
	"name": "A workspace name used to map the default workspace to a named remote workspace.\n" +
		"When configured only the default workspace can be used. This option conflicts\n" +
		"with \"prefix\"",
	"prefix": "A prefix used to filter workspaces using a single configuration. New workspaces\n" +
		"will automatically be prefixed with this prefix. If omitted only the default\n" +
		"workspace can be used. This option conflicts with \"name\"",
}
