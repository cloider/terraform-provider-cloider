package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// defaultTimeout is how long a create, update or delete waits for the resource
// to settle when no timeouts block says otherwise. The CLI's --wait gives 15m.
const defaultTimeout = 20 * time.Minute

// resourceBase hands each resource the client the provider configured.
type resourceBase struct {
	client *client.Client
}

func (b *resourceBase) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	b.client = c
}

type dataSourceBase struct {
	client *client.Client
}

func (b *dataSourceBase) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	b.client = c
}

// findIn lists path and picks the item with the id. The api has no GET by id for
// clusters, bucket keys and registry tokens.
func findIn[T any](ctx context.Context, c *client.Client, path string, idOf func(T) string, id string) (*T, error) {
	var items []T
	if err := c.Get(ctx, path, nil, &items); err != nil {
		return nil, err
	}
	i := slices.IndexFunc(items, func(item T) bool { return idOf(item) == id })
	if i < 0 {
		return nil, nil
	}
	return &items[i], nil
}

func addAPIError(diags *diag.Diagnostics, summary string, err error) {
	diags.AddError(summary, err.Error())
}

// addCreateError is addAPIError for a create, which the api may have done after
// the ingress gave up waiting for it, 10s in. Terraform doesn't know it then.
func addCreateError(diags *diag.Diagnostics, summary string, err error) {
	detail := err.Error()
	if client.MayHaveGoneThrough(err) {
		detail += "\n\nThe api may have created it anyway after the ingress gave up waiting, check the dashboard for one to import or delete."
	}
	diags.AddError(summary, detail)
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func timeString(t time.Time) types.String {
	if t.IsZero() {
		return types.StringNull()
	}
	return types.StringValue(t.UTC().Format(time.RFC3339))
}

// setOrNull is the set of ids a key or token is scoped to, null when there are
// none, so an unset attribute reads back unset.
func setOrNull(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, values)
}

func stringsOf(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	values := []string{}
	if set.IsNull() || set.IsUnknown() {
		return values, nil
	}
	diags := set.ElementsAs(ctx, &values, false)
	return values, diags
}

// paused is whether the status means paused, or on its way there.
func paused(status string) bool {
	return status == "Paused" || status == "Pausing"
}

// setPaused pauses or resumes what basePath names and waits for it to settle.
func setPaused(ctx context.Context, c *client.Client, basePath string, pause bool, what string, diags *diag.Diagnostics) bool {
	action, want := "resume", "Ready"
	if pause {
		action, want = "pause", "Paused"
	}
	if err := c.Post(ctx, basePath+"/"+action, nil, nil); err != nil {
		addAPIError(diags, "Can't "+action+" "+what, err)
		return false
	}
	if _, err := c.WaitStatus(ctx, basePath+"/status", want); err != nil {
		diags.AddError(what+" didn't "+action, err.Error())
		return false
	}
	return true
}

// deleteAndWait deletes what basePath names and waits until it is Terminated.
// The status comes first: a DELETE on a Terminated cluster or database queues
// its teardown again, and buckets and registries refuse one while Terminating.
func deleteAndWait(ctx context.Context, c *client.Client, basePath, what string, diags *diag.Diagnostics) {
	status, err := c.GetText(ctx, basePath+"/status", nil)
	if client.IsNotFound(err) {
		return
	}
	if err != nil {
		addAPIError(diags, "Can't read the status of "+what, err)
		return
	}

	switch strings.TrimSpace(status) {
	case "Terminated":
		return
	case "Terminating":
	default:
		if err := c.Delete(ctx, basePath, nil, nil); err != nil {
			if client.IsNotFound(err) {
				return
			}
			addAPIError(diags, "Can't delete "+what, err)
			return
		}
	}

	if _, err := c.WaitStatus(ctx, basePath+"/status", "Terminated"); err != nil {
		diags.AddError(what+" wasn't deleted", err.Error()+"\n\nRun destroy again to retry.")
	}
}

// adoptWait is how long a create the ingress gave up on may still take: the api
// gives the resource service 3 minutes for a bucket or registry project.
const adoptWait = 4 * time.Minute

// adoptAfterTimeout finds what a create made after the ingress stopped waiting
// for it, 10s in. Buckets and registry projects are created while the api
// answers, and their names are their own across all accounts, so the one of
// that name in the list is the one the create made.
func adoptAfterTimeout[T any](ctx context.Context, c *client.Client, listPath string, match func(T) bool, idAndStatus func(T) (string, string)) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, adoptWait)
	defer cancel()

	var id string
	_, err := client.Poll(ctx, func(ctx context.Context) (string, bool, error) {
		var items []T
		if err := c.Get(ctx, listPath, nil, &items); err != nil {
			return "", false, err
		}
		i := slices.IndexFunc(items, match)
		if i < 0 {
			return "", false, nil
		}
		itemID, status := idAndStatus(items[i])
		switch status {
		case "Ready":
			id = itemID
			return status, true, nil
		case "Failed", "Terminated":
			return status, false, fmt.Errorf("it ended %s, check the dashboard for the reason", status)
		}
		return status, false, nil
	})
	if err != nil && ctx.Err() != nil {
		return "", fmt.Errorf("the ingress gave up on the create 10s in, and it didn't show up in the %s after", adoptWait)
	}
	return id, err
}

// errOrGone explains a read back that failed after a write.
func errOrGone(err error) string {
	if err != nil {
		return err.Error()
	}
	return "The api doesn't have it any more, it may have been deleted in the meantime."
}
