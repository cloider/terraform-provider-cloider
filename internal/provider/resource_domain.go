package provider

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure   = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

type domainResource struct {
	resourceBase
}

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
}

// The api keeps names lower case without the trailing dot, so a name written
// otherwise would never read back the same.
var domainName = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "A free subdomain under one of Cloider's roots, such as `my-app.cloider.app`, or a domain you own. Point a domain you own at the nameservers in `cloider_options`, then add its records with `cloider_dns_record`. Deleting a domain also deletes its DNS records and the routes to it.",
		Attributes: map[string]schema.Attribute{
			"id": computedString("The domain name."),
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name, lower case and without a trailing dot. A name under one of `cloider_options`' `subdomain_roots` is claimed as a free subdomain, anything else is added as a custom domain. Changing it replaces the domain.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(domainName, "must be lower case, without a trailing dot"),
				},
			},
			"type": computedString("`Subdomain` or `CustomDomain`."),
			"status": schema.StringAttribute{
				MarkdownDescription: "`Pending` until something routes to it, `Active` while something does, `Suspended` once nothing does any more.",
				Computed:            true,
			},
			"created_at": computedString("When the domain was added, RFC 3339."),
		},
	}
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()

	// Adding a domain the account already has answers with it, so without this a
	// destroy would delete records and routes this configuration never made
	var existing client.Domain
	err := r.client.Get(ctx, "/domains/"+url.PathEscape(name), nil, &existing)
	if err == nil {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Domain already added",
			name+" is already one of your domains. Import it with `terraform import <address> "+name+"` to manage it here.")
		return
	}
	if !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't check for domain "+name, err)
		return
	}

	var options client.Options
	if err := r.client.Get(ctx, "/options", nil, &options); err != nil {
		addAPIError(&resp.Diagnostics, "Can't read the subdomain roots", err)
		return
	}
	domainType := "CustomDomain"
	for _, root := range options.SubdomainRoots {
		if strings.HasSuffix(name, "."+root) {
			domainType = "Subdomain"
		}
	}

	var created client.Domain
	if err := r.client.Post(ctx, "/domains", map[string]string{"domain_name": name, "domain_type": domainType}, &created); err != nil {
		addCreateError(&resp.Diagnostics, "Can't add domain "+name, err)
		return
	}
	fillDomain(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var d client.Domain
	err := r.client.Get(ctx, "/domains/"+url.PathEscape(state.ID.ValueString()), nil, &d)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read domain "+state.ID.ValueString(), err)
		return
	}
	fillDomain(&state, d)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update is never called with a change, every attribute replaces the domain.
func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/domains/"+url.PathEscape(state.ID.ValueString()), nil, nil); err != nil && !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't delete domain "+state.ID.ValueString(), err)
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(req.ID)), ".")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

func fillDomain(m *domainModel, d client.Domain) {
	m.ID = types.StringValue(d.DomainName)
	m.Name = types.StringValue(d.DomainName)
	m.Type = types.StringValue(d.DomainType)
	m.Status = types.StringValue(d.Status)
	m.CreatedAt = timeString(d.CreatedAt)
}
