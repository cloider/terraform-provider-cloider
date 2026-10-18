package provider

import (
	"context"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure   = &clusterResource{}
	_ resource.ResourceWithImportState = &clusterResource{}
)

type clusterResource struct {
	resourceBase
}

func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

type clusterModel struct {
	ID         types.String   `tfsdk:"id"`
	Name       types.String   `tfsdk:"name"`
	Version    types.String   `tfsdk:"version"`
	Paused     types.Bool     `tfsdk:"paused"`
	Label      types.String   `tfsdk:"label"`
	Status     types.String   `tfsdk:"status"`
	CreatedAt  types.String   `tfsdk:"created_at"`
	KubeConfig types.Object   `tfsdk:"kube_config"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	kubeString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, Sensitive: true}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "A managed Kubernetes cluster. Creating one waits until it is `Ready`. There is no rename or upgrade, so changing `name` or `version` replaces the cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The cluster's id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The cluster's name: lower case letters, digits and `-`, starting with a letter. Names don't have to be unique, the dashboard tells clusters of the same name apart by `label`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The Kubernetes version, one of `cloider_options`' `cluster_versions`. Defaults to the newest offered.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Whether the cluster is paused. A paused cluster keeps its state and isn't billed for compute. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "The name the dashboard shows, `name` with a `-1`, `-2`... suffix when another cluster has it.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "`Creating`, `Starting`, `Ready`, `Pausing`, `Paused` or `Terminating`.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the cluster was created, RFC 3339.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kube_config": schema.SingleNestedAttribute{
				MarkdownDescription: "How to reach the cluster's Kubernetes api, for the `kubernetes` and `helm` providers.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"raw":                    kubeString("The whole kubeconfig file."),
					"host":                   kubeString("The Kubernetes api's url."),
					"cluster_ca_certificate": kubeString("The PEM CA certificate the api's certificate is signed by."),
					"client_certificate":     kubeString("The PEM client certificate to authenticate with."),
					"client_key":             kubeString("The PEM key of `client_certificate`."),
					"token":                  kubeString("The bearer token to authenticate with, when the kubeconfig uses one."),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
		},
	}
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterModel
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

	version := plan.Version.ValueString()
	if version == "" {
		var options client.Options
		if err := r.client.Get(ctx, "/options", nil, &options); err != nil {
			addAPIError(&resp.Diagnostics, "Can't read the cluster versions offered", err)
			return
		}
		if len(options.ClusterVersions) == 0 {
			resp.Diagnostics.AddError("No Kubernetes version is offered", "The api offers no cluster version at the moment.")
			return
		}
		version = options.ClusterVersions[0]
	}

	var created client.Created
	body := map[string]any{"cluster_name": plan.Name.ValueString(), "cluster_version": version, "is_ha": false}
	if err := r.client.Post(ctx, "/clusters", body, &created); err != nil {
		addCreateError(&resp.Diagnostics, "Can't create the cluster", err)
		return
	}

	// In state before waiting: a wait that fails leaves the cluster tainted rather
	// than unknown to Terraform, and names aren't unique, so a retry would make a second
	state := plan
	state.ID = types.StringValue(created.HashedId)
	state.Version = types.StringValue(version)
	state.Paused = types.BoolValue(false)
	state.Label = types.StringNull()
	state.Status = types.StringValue("Creating")
	state.CreatedAt = types.StringNull()
	state.KubeConfig = types.ObjectNull(kubeConfigAttrTypes)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.WaitStatus(ctx, "/clusters/"+created.HashedId+"/status", "Ready"); err != nil {
		resp.Diagnostics.AddError("Cluster "+plan.Name.ValueString()+" didn't become ready", err.Error())
		return
	}
	if plan.Paused.ValueBool() {
		if !setPaused(ctx, r.client, "/clusters/"+created.HashedId, true, "cluster "+plan.Name.ValueString(), &resp.Diagnostics) {
			return
		}
	}

	found, err := r.read(ctx, &state)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the cluster back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.read(ctx, &state)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read cluster "+state.ID.ValueString(), err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if plan.Paused.ValueBool() != state.Paused.ValueBool() {
		if !setPaused(ctx, r.client, "/clusters/"+state.ID.ValueString(), plan.Paused.ValueBool(), "cluster "+state.Name.ValueString(), &resp.Diagnostics) {
			return
		}
	}

	plan.ID = state.ID
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the cluster back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterModel
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

	deleteAndWait(ctx, r.client, "/clusters/"+state.ID.ValueString(), "cluster "+state.Name.ValueString(), &resp.Diagnostics)
}

func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read fills m from the api, false when the cluster is gone.
func (r *clusterResource) read(ctx context.Context, m *clusterModel) (bool, error) {
	id := m.ID.ValueString()
	c, err := findIn(ctx, r.client, "/clusters", func(c client.Cluster) string { return c.HashedId }, id)
	if err != nil {
		return false, err
	}
	if c == nil || c.Status == "Terminated" {
		return false, nil
	}

	m.Name = types.StringValue(c.ClusterName)
	m.Version = types.StringValue(c.ClusterVersion)
	m.Label = types.StringValue(c.ClusterLabel)
	m.Status = types.StringValue(c.Status)
	m.Paused = types.BoolValue(paused(c.Status))
	m.CreatedAt = timeString(c.CreatedAt)

	kubeconfig, err := r.client.GetText(ctx, "/clusters/"+id+"/kubeconfig", nil)
	if client.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	m.KubeConfig = kubeConfigValue(kubeconfig)
	return true, nil
}
