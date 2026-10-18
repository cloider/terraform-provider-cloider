package provider

import (
	"context"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure   = &registryResource{}
	_ resource.ResourceWithImportState = &registryResource{}
)

type registryResource struct {
	resourceBase
}

func NewRegistryResource() resource.Resource {
	return &registryResource{}
}

type registryModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Public         types.Bool     `tfsdk:"public"`
	StorageLimitGi types.Int64    `tfsdk:"storage_limit_gi"`
	ImmutableTags  types.Bool     `tfsdk:"immutable_tags"`
	Status         types.String   `tfsdk:"status"`
	Host           types.String   `tfsdk:"host"`
	URL            types.String   `tfsdk:"url"`
	CreatedAt      types.String   `tfsdk:"created_at"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func (r *registryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry"
}

func (r *registryResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "A container registry project, where images are pushed to and pulled from as `<url>/<repository>:<tag>`. Push and pull with a `cloider_registry_token`.",
		Attributes: map[string]schema.Attribute{
			"id": computedString("The project's id."),
			"name": schema.StringAttribute{
				MarkdownDescription: "The project's name, 3 to 63 lower case letters, digits and `-`. Project names are shared by every Cloider account, so it has to be one nobody else has. Changing it replaces the project and its images.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "Whether anyone may pull its images without a token. Pushing always needs one. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"storage_limit_gi": schema.Int64Attribute{
				MarkdownDescription: "The most GiB its images may take, 1 to 20. A push past it is refused. Defaults to `5`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(5),
				Validators:          []validator.Int64{int64validator.Between(1, 20)},
			},
			"immutable_tags": schema.BoolAttribute{
				MarkdownDescription: "Whether pushing over a tag or deleting one is refused, `latest` included. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "`Ready`, `Failed` or `Terminating`.",
				Computed:            true,
			},
			"host":       computedString("The registry host to `docker login` to."),
			"url":        computedString("The prefix of every image in the project, `<host>/<name>`."),
			"created_at": computedString("When the project was created, RFC 3339."),
			"timeouts":   timeouts.Attributes(ctx, timeouts.Opts{Create: true, Delete: true}),
		},
	}
}

func (r *registryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan registryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name := plan.Name.ValueString()
	body := map[string]any{
		"name":             name,
		"public":           plan.Public.ValueBool(),
		"storage_limit_gi": plan.StorageLimitGi.ValueInt64(),
		"immutable_tags":   plan.ImmutableTags.ValueBool(),
	}
	var created client.Created
	err := r.client.Post(ctx, "/registry/projects", body, &created)
	if client.MayHaveGoneThrough(err) {
		created.HashedId, err = adoptAfterTimeout(ctx, r.client, "/registry/projects",
			func(p client.Registry) bool { return p.Name == name && !p.Apps },
			func(p client.Registry) (string, string) { return p.HashedId, p.Status })
	}
	if err != nil {
		resp.Diagnostics.AddError("Can't create registry project "+name, err.Error())
		return
	}

	plan.ID = types.StringValue(created.HashedId)
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the registry project back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *registryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state registryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.read(ctx, &state)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read registry project "+state.ID.ValueString(), err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *registryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state registryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	basePath := "/registry/projects/" + state.ID.ValueString()
	what := "registry project " + state.Name.ValueString()
	// The api takes both settings at once
	if !plan.Public.Equal(state.Public) || !plan.StorageLimitGi.Equal(state.StorageLimitGi) {
		settings := client.RegistrySettings{Public: plan.Public.ValueBool(), StorageLimitGi: int(plan.StorageLimitGi.ValueInt64())}
		if err := r.client.Put(ctx, basePath+"/settings", settings, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change the settings of "+what, err)
			return
		}
	}
	// Its own call, as the setting is one of the project's immutability rules
	if !plan.ImmutableTags.Equal(state.ImmutableTags) {
		if err := r.client.Put(ctx, basePath+"/immutable-tags", map[string]bool{"immutable_tags": plan.ImmutableTags.ValueBool()}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change the immutable tags of "+what, err)
			return
		}
	}

	plan.ID = state.ID
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the registry project back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *registryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state registryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := state.Timeouts.Delete(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deleteAndWait(ctx, r.client, "/registry/projects/"+state.ID.ValueString(), "registry project "+state.Name.ValueString(), &resp.Diagnostics)
}

func (r *registryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var p client.Registry
	if err := r.client.Get(ctx, "/registry/projects/"+req.ID, nil, &p); err != nil {
		addAPIError(&resp.Diagnostics, "Can't read registry project "+req.ID, err)
		return
	}
	if p.Apps {
		resp.Diagnostics.AddError("Can't manage the apps project", "Project "+p.Name+" is the one Cloider builds your apps' images into. It is made and removed with your apps, so Terraform can't manage it.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read fills m from the api, false when the project is gone.
func (r *registryResource) read(ctx context.Context, m *registryModel) (bool, error) {
	basePath := "/registry/projects/" + m.ID.ValueString()
	var p client.Registry
	err := r.client.Get(ctx, basePath, nil, &p)
	if client.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if p.Status == "Terminated" {
		return false, nil
	}

	var immutability client.RegistryImmutability
	if err := r.client.Get(ctx, basePath+"/immutability", nil, &immutability); err != nil {
		return false, err
	}

	m.Name = types.StringValue(p.Name)
	m.Public = types.BoolValue(p.Public)
	m.StorageLimitGi = types.Int64Value(int64(p.StorageLimitGi))
	m.ImmutableTags = types.BoolValue(immutability.ImmutableTags)
	m.Status = types.StringValue(p.Status)
	m.Host = stringOrNull(p.Host)
	m.URL = stringOrNull(p.URL)
	m.CreatedAt = timeString(p.CreatedAt)
	return true, nil
}
