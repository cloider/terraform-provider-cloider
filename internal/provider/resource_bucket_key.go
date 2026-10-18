package provider

import (
	"context"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure      = &bucketKeyResource{}
	_ resource.ResourceWithImportState    = &bucketKeyResource{}
	_ resource.ResourceWithValidateConfig = &bucketKeyResource{}
)

type bucketKeyResource struct {
	resourceBase
}

func NewBucketKeyResource() resource.Resource {
	return &bucketKeyResource{}
}

type bucketKeyModel struct {
	ID         types.String `tfsdk:"id"`
	Label      types.String `tfsdk:"label"`
	AllBuckets types.Bool   `tfsdk:"all_buckets"`
	Buckets    types.Set    `tfsdk:"buckets"`
	ReadOnly   types.Bool   `tfsdk:"read_only"`
	AccessKey  types.String `tfsdk:"access_key"`
	SecretKey  types.String `tfsdk:"secret_key"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func (r *bucketKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_key"
}

func (r *bucketKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "An S3 access key for some or all of your buckets. The secret key is only ever shown when the key is created, so an imported key has no `secret_key`.",
		Attributes: map[string]schema.Attribute{
			"id": computedString("The key's id."),
			"label": schema.StringAttribute{
				MarkdownDescription: "What the key is for, up to 64 characters.",
				Required:            true,
			},
			"all_buckets": schema.BoolAttribute{
				MarkdownDescription: "Whether the key reaches every bucket, those made later included. Set this or `buckets`. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"buckets": schema.SetAttribute{
				MarkdownDescription: "The ids of the buckets the key reaches. Set this or `all_buckets`. A bucket that is deleted drops out of it.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"read_only": schema.BoolAttribute{
				MarkdownDescription: "Whether the key may only read. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"access_key": computedString("The access key id, `AWS_ACCESS_KEY_ID`."),
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "The secret access key, `AWS_SECRET_ACCESS_KEY`. Only known for a key this configuration created.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": computedString("When the key was created, RFC 3339."),
		},
	}
}

func (r *bucketKeyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var all types.Bool
	var buckets types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("all_buckets"), &all)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("buckets"), &buckets)...)
	if resp.Diagnostics.HasError() || all.IsUnknown() || buckets.IsUnknown() {
		return
	}
	switch {
	case all.ValueBool() && !buckets.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("buckets"), "Conflicting scope", "all_buckets and buckets don't go together.")
	case !all.ValueBool() && buckets.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("buckets"), "Missing scope", "Set buckets, or all_buckets = true.")
	}
}

func (r *bucketKeyResource) form(ctx context.Context, m bucketKeyModel) (client.BucketKeyForm, diag.Diagnostics) {
	buckets, diags := stringsOf(ctx, m.Buckets)
	return client.BucketKeyForm{
		Label:      m.Label.ValueString(),
		AllBuckets: m.AllBuckets.ValueBool(),
		Buckets:    buckets,
		ReadOnly:   m.ReadOnly.ValueBool(),
	}, diags
}

func (r *bucketKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	form, diags := r.form(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.CreatedBucketKey
	if err := r.client.Post(ctx, "/bucket-keys", form, &created); err != nil {
		addCreateError(&resp.Diagnostics, "Can't create the bucket key", err)
		return
	}

	plan.ID = types.StringValue(created.Key.HashedId)
	plan.SecretKey = types.StringValue(created.SecretKey)
	resp.Diagnostics.Append(r.fill(ctx, &plan, created.Key)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bucketKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := findIn(ctx, r.client, "/bucket-keys", func(k client.BucketKey) string { return k.HashedId }, state.ID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read bucket key "+state.ID.ValueString(), err)
		return
	}
	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(r.fill(ctx, &state, *key)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *bucketKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state bucketKeyModel
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
	if err := r.client.Put(ctx, "/bucket-keys/"+id, form, nil); err != nil {
		addAPIError(&resp.Diagnostics, "Can't update bucket key "+id, err)
		return
	}
	key, err := findIn(ctx, r.client, "/bucket-keys", func(k client.BucketKey) string { return k.HashedId }, id)
	if err != nil || key == nil {
		resp.Diagnostics.AddError("Can't read the bucket key back", errOrGone(err))
		return
	}

	plan.ID = state.ID
	plan.SecretKey = state.SecretKey
	resp.Diagnostics.Append(r.fill(ctx, &plan, *key)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bucketKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/bucket-keys/"+state.ID.ValueString(), nil, nil); err != nil && !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't delete bucket key "+state.ID.ValueString(), err)
	}
}

func (r *bucketKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *bucketKeyResource) fill(ctx context.Context, m *bucketKeyModel, key client.BucketKey) diag.Diagnostics {
	ids := key.Buckets
	if key.AllBuckets {
		ids = nil
	}
	buckets, diags := setOrNull(ctx, ids)
	m.Label = types.StringValue(key.Label)
	m.AllBuckets = types.BoolValue(key.AllBuckets)
	m.Buckets = buckets
	m.ReadOnly = types.BoolValue(key.ReadOnly)
	m.AccessKey = types.StringValue(key.AccessKey)
	m.CreatedAt = timeString(key.CreatedAt)
	return diags
}
