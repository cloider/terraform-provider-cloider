package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.ResourceWithConfigure   = &domainVerificationResource{}
	_ resource.ResourceWithImportState = &domainVerificationResource{}
)

// defaultVerifyTimeout is how long a create waits for the TXT record to be found
const defaultVerifyTimeout = 10 * time.Minute

// verifyInterval spaces the checks out: each one makes the api query the
// domain's nameservers, and all of them count against its rate limit.
var verifyInterval = 15 * time.Second

type domainVerificationResource struct {
	resourceBase
}

func NewDomainVerificationResource() resource.Resource {
	return &domainVerificationResource{}
}

type domainVerificationModel struct {
	ID       types.String   `tfsdk:"id"`
	Domain   types.String   `tfsdk:"domain"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *domainVerificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_verification"
}

func (r *domainVerificationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Verifies a custom `cloider_domain` is yours. Creating it waits until Cloider finds the domain's TXT record, from its `verification_record_name` and `verification_record_value`, checking every 15 seconds. Reference it from a `cloider_target` so the target is created once the domain is verified. Deleting it leaves the domain verified.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain to verify, a `cloider_domain`'s `name`. A domain that is already verified, such as a free subdomain, is done at once. Changing it replaces the verification.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(domainName, "must be lower case, without a trailing dot"),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func (r *domainVerificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainVerificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultVerifyTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name := plan.Domain.ValueString()
	if err := verifyDomain(ctx, r.client, name); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain"), "Domain "+name+" isn't verified", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// verifyDomain has the api check the domain's TXT record until it is found or ctx
// ends. A record added moments ago may not be answered yet, so every failure is
// retried except a domain that isn't the account's or a token that can't verify.
func verifyDomain(ctx context.Context, c *client.Client, name string) error {
	var last error
	for {
		err := c.Post(ctx, "/domains/"+url.PathEscape(name)+"/verify", nil, nil)
		if err == nil {
			return nil
		}
		switch client.StatusOf(err) {
		case http.StatusNotFound:
			return fmt.Errorf("%s isn't one of your domains, add it with a cloider_domain first", name)
		case http.StatusUnauthorized, http.StatusForbidden:
			return err
		}
		// The api's own answer says what is missing, a later timeout or rate limit doesn't
		if last == nil || client.StatusOf(err) == http.StatusBadRequest {
			last = err
		}
		tflog.Info(ctx, "domain not verified yet, checking again", map[string]any{"domain": name, "reason": err.Error()})

		select {
		case <-ctx.Done():
			return errors.New(last.Error() + "\n\nGave up waiting. Apply again once the record is in place, or raise the timeout with a timeouts block.")
		case <-time.After(verifyInterval):
		}
	}
}

func (r *domainVerificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainVerificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var d client.Domain
	err := r.client.Get(ctx, "/domains/"+url.PathEscape(state.Domain.ValueString()), nil, &d)
	// A domain deleted, or deleted and added back unverified, needs verifying again
	if client.IsNotFound(err) || (err == nil && !d.Verified) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read domain "+state.Domain.ValueString(), err)
		return
	}
	state.ID = types.StringValue(d.DomainName)
	state.Domain = types.StringValue(d.DomainName)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update only ever changes timeouts, a new domain replaces the verification.
func (r *domainVerificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainVerificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete only forgets it: the api can't unverify a domain, it stays verified until
// the domain itself is deleted.
func (r *domainVerificationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *domainVerificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(req.ID)), ".")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), name)...)
}
