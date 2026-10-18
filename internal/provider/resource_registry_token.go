package provider

import (
	"context"
	"math"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure      = &registryTokenResource{}
	_ resource.ResourceWithImportState    = &registryTokenResource{}
	_ resource.ResourceWithValidateConfig = &registryTokenResource{}
)

type registryTokenResource struct {
	resourceBase
}

func NewRegistryTokenResource() resource.Resource {
	return &registryTokenResource{}
}

type registryTokenModel struct {
	ID            types.String `tfsdk:"id"`
	Label         types.String `tfsdk:"label"`
	AllProjects   types.Bool   `tfsdk:"all_projects"`
	Projects      types.Set    `tfsdk:"projects"`
	ReadOnly      types.Bool   `tfsdk:"read_only"`
	ExpiresInDays types.Int64  `tfsdk:"expires_in_days"`
	Username      types.String `tfsdk:"username"`
	Secret        types.String `tfsdk:"secret"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *registryTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_token"
}

func (r *registryTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "A token to push and pull images of some or all of your registry projects, `docker login <host> -u <username> -p <secret>`. The secret is only ever shown when the token is created, so an imported token has no `secret`.",
		Attributes: map[string]schema.Attribute{
			"id": computedString("The token's id."),
			"label": schema.StringAttribute{
				MarkdownDescription: "What the token is for.",
				Required:            true,
			},
			"all_projects": schema.BoolAttribute{
				MarkdownDescription: "Whether the token reaches every project, those made later included. Set this or `projects`. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"projects": schema.SetAttribute{
				MarkdownDescription: "The ids of the `cloider_registry` projects the token reaches. Set this or `all_projects`.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"read_only": schema.BoolAttribute{
				MarkdownDescription: "Whether the token may only pull. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"expires_in_days": schema.Int64Attribute{
				MarkdownDescription: "Days until the token stops working, 1 to 3650, or `0` for never. Expiry can't be changed, so changing it replaces the token. Defaults to `0`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators:          []validator.Int64{int64validator.Between(0, 3650)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"username": computedString("The username to log in with."),
			"secret": schema.StringAttribute{
				MarkdownDescription: "The password to log in with. Only known for a token this configuration created.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expires_at": computedString("When the token stops working, RFC 3339, null for never."),
			"created_at": computedString("When the token was created, RFC 3339."),
		},
	}
}

func (r *registryTokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var all types.Bool
	var projects types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("all_projects"), &all)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("projects"), &projects)...)
	if resp.Diagnostics.HasError() || all.IsUnknown() || projects.IsUnknown() {
		return
	}
	switch {
	case all.ValueBool() && !projects.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("projects"), "Conflicting scope", "all_projects and projects don't go together.")
	case !all.ValueBool() && projects.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("projects"), "Missing scope", "Set projects, or all_projects = true.")
	}
}

func (r *registryTokenResource) form(ctx context.Context, m registryTokenModel) (client.RegistryTokenForm, diag.Diagnostics) {
	projects, diags := stringsOf(ctx, m.Projects)
	return client.RegistryTokenForm{
		Label:       m.Label.ValueString(),
		AllProjects: m.AllProjects.ValueBool(),
		Projects:    projects,
		ReadOnly:    m.ReadOnly.ValueBool(),
	}, diags
}

func (r *registryTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan registryTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	form, diags := r.form(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.CreatedRegistryToken
	body := client.NewRegistryTokenForm{RegistryTokenForm: form, ExpiresInDays: int(plan.ExpiresInDays.ValueInt64())}
	if err := r.client.Post(ctx, "/registry/tokens", body, &created); err != nil {
		addCreateError(&resp.Diagnostics, "Can't create the registry token", err)
		return
	}

	plan.ID = types.StringValue(created.Token.HashedId)
	plan.Secret = types.StringValue(created.Secret)
	resp.Diagnostics.Append(r.fill(ctx, &plan, created.Token)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *registryTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state registryTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := findIn(ctx, r.client, "/registry/tokens", func(t client.RegistryToken) string { return t.HashedId }, state.ID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read registry token "+state.ID.ValueString(), err)
		return
	}
	if token == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(r.fill(ctx, &state, *token)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *registryTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state registryTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	form, diags := r.form(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.Put(ctx, "/registry/tokens/"+id, form, nil); err != nil {
		addAPIError(&resp.Diagnostics, "Can't update registry token "+id, err)
		return
	}
	token, err := findIn(ctx, r.client, "/registry/tokens", func(t client.RegistryToken) string { return t.HashedId }, id)
	if err != nil || token == nil {
		resp.Diagnostics.AddError("Can't read the registry token back", errOrGone(err))
		return
	}

	plan.ID = state.ID
	plan.Secret = state.Secret
	resp.Diagnostics.Append(r.fill(ctx, &plan, *token)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *registryTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state registryTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/registry/tokens/"+state.ID.ValueString(), nil, nil); err != nil && !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't delete registry token "+state.ID.ValueString(), err)
	}
}

func (r *registryTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *registryTokenResource) fill(ctx context.Context, m *registryTokenModel, token client.RegistryToken) diag.Diagnostics {
	ids := token.Projects
	if token.AllProjects {
		ids = nil
	}
	projects, diags := setOrNull(ctx, ids)
	m.Label = types.StringValue(token.Label)
	m.AllProjects = types.BoolValue(token.AllProjects)
	m.Projects = projects
	m.ReadOnly = types.BoolValue(token.ReadOnly)
	m.Username = types.StringValue(token.Username)
	m.CreatedAt = timeString(token.CreatedAt)
	m.ExpiresAt = types.StringNull()
	if token.ExpiresAt != nil {
		m.ExpiresAt = timeString(*token.ExpiresAt)
	}

	// The api keeps when it expires, not the days asked for, so only an import
	// works them out. The rounding takes up the moments between the two.
	if m.ExpiresInDays.IsNull() || m.ExpiresInDays.IsUnknown() {
		days := int64(0)
		if token.ExpiresAt != nil {
			days = int64(math.Round(token.ExpiresAt.Sub(token.CreatedAt).Hours() / 24))
		}
		m.ExpiresInDays = types.Int64Value(days)
	}
	return diags
}
