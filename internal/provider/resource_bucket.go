package provider

import (
	"context"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure   = &bucketResource{}
	_ resource.ResourceWithImportState = &bucketResource{}
)

type bucketResource struct {
	resourceBase
}

func NewBucketResource() resource.Resource {
	return &bucketResource{}
}

type bucketModel struct {
	ID            types.String   `tfsdk:"id"`
	Name          types.String   `tfsdk:"name"`
	Region        types.String   `tfsdk:"region"`
	Public        types.Bool     `tfsdk:"public"`
	Versioning    types.Bool     `tfsdk:"versioning"`
	Website       types.Bool     `tfsdk:"website"`
	IndexDocument types.String   `tfsdk:"index_document"`
	ErrorDocument types.String   `tfsdk:"error_document"`
	Status        types.String   `tfsdk:"status"`
	Endpoint      types.String   `tfsdk:"endpoint"`
	URL           types.String   `tfsdk:"url"`
	WebsiteURL    types.String   `tfsdk:"website_url"`
	CreatedAt     types.String   `tfsdk:"created_at"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "An S3 compatible object storage bucket. Access it with a `cloider_bucket_key`.",
		Attributes: map[string]schema.Attribute{
			"id": computedString("The bucket's id."),
			"name": schema.StringAttribute{
				MarkdownDescription: "The bucket's name, 3 to 63 lower case letters, digits and `-`. Bucket names are shared by every Cloider account, so it has to be one nobody else has. Changing it replaces the bucket.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region code, see `cloider_bucket_regions`. Defaults to the first region. Changing it replaces the bucket.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "Whether anyone may read the bucket's files without a key. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"versioning": schema.BoolAttribute{
				MarkdownDescription: "Whether the bucket keeps previous versions of its files. Versioning that was on can only be suspended, so `false` after `true` keeps the versions already kept. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"website": schema.BoolAttribute{
				MarkdownDescription: "Whether the bucket is served as a static website at `website_url`, and on the domains a `cloider_target` with `bucket_id` routes to it. The website is read without a key, so it needs `public`. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"index_document": schema.StringAttribute{
				MarkdownDescription: "The file the website serves for a path ending in a slash, a name without a slash. Defaults to `index.html`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("index.html"),
			},
			"error_document": schema.StringAttribute{
				MarkdownDescription: "The key the website serves, with a 404, for anything missing. Defaults to `\"\"`, a built-in page.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "`Ready`, `Failed` or `Terminating`.",
				Computed:            true,
			},
			"endpoint": computedString("The S3 endpoint of the bucket's region."),
			"url":      computedString("The bucket's url, where a public bucket's files are read."),
			"website_url": schema.StringAttribute{
				MarkdownDescription: "The website's own address, `https://<name>.s3-website.<region>.<domain>`, while `website` is on.",
				Computed:            true,
			},
			"created_at": computedString("When the bucket was created, RFC 3339."),
			"timeouts":   timeouts.Attributes(ctx, timeouts.Opts{Create: true, Delete: true}),
		},
	}
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketModel
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

	region := plan.Region.ValueString()
	if region == "" {
		var regions []client.BucketRegion
		if err := r.client.Get(ctx, "/buckets/regions", nil, &regions); err != nil {
			addAPIError(&resp.Diagnostics, "Can't read the bucket regions", err)
			return
		}
		if len(regions) == 0 {
			resp.Diagnostics.AddError("No bucket region", "The api offers no bucket region at the moment.")
			return
		}
		region = regions[0].Code
	}

	name := plan.Name.ValueString()
	body := map[string]any{"bucket_name": name, "region": region, "public": plan.Public.ValueBool(), "versioning": plan.Versioning.ValueBool(), "website": plan.Website.ValueBool()}
	var created client.Created
	err := r.client.Post(ctx, "/buckets", body, &created)
	if client.MayHaveGoneThrough(err) {
		created.HashedId, err = adoptAfterTimeout(ctx, r.client, "/buckets",
			func(b client.Bucket) bool { return b.BucketName == name },
			func(b client.Bucket) (string, string) { return b.HashedId, b.Status })
	}
	if err != nil {
		resp.Diagnostics.AddError("Can't create bucket "+name, err.Error())
		return
	}

	plan.ID = types.StringValue(created.HashedId)
	// The create takes website alone, the documents start as the defaults
	if plan.IndexDocument.ValueString() != "index.html" || plan.ErrorDocument.ValueString() != "" {
		if err := r.putWebsite(ctx, plan); err != nil {
			// The new bucket still goes into the state, tainted, rather than left behind
			addAPIError(&resp.Diagnostics, "Can't set the website of bucket "+name, err)
		}
	}

	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the bucket back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bucketResource) putWebsite(ctx context.Context, m bucketModel) error {
	form := client.BucketWebsiteForm{
		Enabled:       m.Website.ValueBool(),
		IndexDocument: m.IndexDocument.ValueString(),
		ErrorDocument: m.ErrorDocument.ValueString(),
	}
	return r.client.Put(ctx, "/buckets/"+m.ID.ValueString()+"/website", form, nil)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.read(ctx, &state)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read bucket "+state.ID.ValueString(), err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state bucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	basePath := "/buckets/" + state.ID.ValueString()
	plan.ID = state.ID

	// The website needs the bucket public: it goes off before the bucket turns
	// private, and on after it turns public.
	websiteChanged := !plan.Website.Equal(state.Website) || !plan.IndexDocument.Equal(state.IndexDocument) || !plan.ErrorDocument.Equal(state.ErrorDocument)
	websiteFirst := websiteChanged && !plan.Website.ValueBool()
	setWebsite := func() bool {
		if err := r.putWebsite(ctx, plan); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change the website of bucket "+state.Name.ValueString(), err)
			return false
		}
		return true
	}

	// One at a time: a bucket takes one settings change at once
	if websiteFirst && !setWebsite() {
		return
	}
	if !plan.Public.Equal(state.Public) {
		if err := r.client.Put(ctx, basePath+"/visibility", map[string]bool{"public": plan.Public.ValueBool()}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change the visibility of bucket "+state.Name.ValueString(), err)
			return
		}
	}
	if websiteChanged && !websiteFirst && !setWebsite() {
		return
	}
	if !plan.Versioning.Equal(state.Versioning) {
		if err := r.client.Put(ctx, basePath+"/versioning", map[string]bool{"enabled": plan.Versioning.ValueBool()}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change versioning of bucket "+state.Name.ValueString(), err)
			return
		}
	}

	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the bucket back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketModel
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

	deleteAndWait(ctx, r.client, "/buckets/"+state.ID.ValueString(), "bucket "+state.Name.ValueString(), &resp.Diagnostics)
}

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read fills m from the api, false when the bucket is gone.
func (r *bucketResource) read(ctx context.Context, m *bucketModel) (bool, error) {
	basePath := "/buckets/" + m.ID.ValueString()
	var b client.Bucket
	err := r.client.Get(ctx, basePath, nil, &b)
	if client.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if b.Status == "Terminated" {
		return false, nil
	}

	var versioning client.BucketVersioning
	if err := r.client.Get(ctx, basePath+"/versioning", nil, &versioning); err != nil {
		return false, err
	}

	m.Name = types.StringValue(b.BucketName)
	m.Region = types.StringValue(b.Region)
	m.Public = types.BoolValue(b.Public)
	m.Versioning = types.BoolValue(versioning.Status == "Enabled")
	m.Website = types.BoolValue(b.Website)
	m.IndexDocument = types.StringValue(b.WebsiteIndex)
	m.ErrorDocument = types.StringValue(b.WebsiteError)
	m.Status = types.StringValue(b.Status)
	m.Endpoint = stringOrNull(b.Endpoint)
	m.URL = stringOrNull(b.URL)
	m.WebsiteURL = stringOrNull(b.WebsiteURL)
	m.CreatedAt = timeString(b.CreatedAt)
	return true, nil
}
