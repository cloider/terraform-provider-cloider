package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure        = &targetResource{}
	_ resource.ResourceWithImportState      = &targetResource{}
	_ resource.ResourceWithConfigValidators = &targetResource{}
)

// targetPath is what the api allows in a path
var targetPath = regexp.MustCompile(`^/[A-Za-z0-9/_.-]*$`)

type targetResource struct {
	resourceBase
}

func NewTargetResource() resource.Resource {
	return &targetResource{}
}

type targetModel struct {
	ID         types.String `tfsdk:"id"`
	Domain     types.String `tfsdk:"domain"`
	Path       types.String `tfsdk:"path"`
	TargetPath types.String `tfsdk:"target_path"`
	ClusterID  types.String `tfsdk:"cluster_id"`
	Namespace  types.String `tfsdk:"namespace"`
	Service    types.String `tfsdk:"service"`
	Port       types.Int64  `tfsdk:"port"`
	Protocol   types.String `tfsdk:"protocol"`
	AppID      types.String `tfsdk:"app_id"`
	BucketID   types.String `tfsdk:"bucket_id"`
	// <namespace>/<service> of a LoadBalancer target
	Loadbalancer   types.String `tfsdk:"loadbalancer"`
	LoadbalancerIP types.String `tfsdk:"loadbalancer_ip"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (r *targetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (r *targetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	pathValidator := stringvalidator.RegexMatches(targetPath, "must start with / and hold only letters, digits and / _ . -")
	clusterOnly := func(description string) string {
		return description + " Set with `cluster_id`."
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Routes a path of a `cloider_domain` to a Service in a `cloider_cluster`, to an app, or to a `cloider_bucket`'s static website, over HTTPS. This is how a cluster's Services are reached from the internet: Ingress objects in a cluster aren't served. A custom domain can instead point straight at a LoadBalancer Service in a cluster, with an A record to its IP, for any port and protocol.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The target's id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain to route, a free subdomain or a custom domain. Changing it replaces the target.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path on the domain to route, and everything under it. A domain routes each path to one target. Defaults to `/`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/"),
				Validators:          []validator.String{pathValidator},
			},
			"target_path": schema.StringAttribute{
				MarkdownDescription: "The path a request reaches the target with, in place of `path`. Defaults to `path`, so requests arrive as they were made.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{pathValidator},
				PlanModifiers:       []planmodifier.String{targetPathDefault{}},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The id of the `cloider_cluster` whose Service to route to. Set this, with `namespace`, `service` and `port` or with `loadbalancer`, or `app_id`, or `bucket_id`.",
				Optional:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: clusterOnly("The Service's namespace."),
				Optional:            true,
				Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("cluster_id"))},
			},
			"service": schema.StringAttribute{
				MarkdownDescription: clusterOnly("The Service's name."),
				Optional:            true,
				Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("cluster_id"))},
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: clusterOnly("The Service's port to send requests to, which speaks plain HTTP."),
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
					int64validator.AlsoRequires(path.MatchRoot("cluster_id")),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol of the Service's port, `TCP` or `UDP`. Defaults to `TCP`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("TCP", "UDP"),
					stringvalidator.ConflictsWith(path.MatchRoot("app_id"), path.MatchRoot("bucket_id"), path.MatchRoot("loadbalancer")),
				},
				PlanModifiers: []planmodifier.String{protocolPlan{}},
			},
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The id of the app to route to, see `cloider app list -o wide`. Set this, `bucket_id` or `cluster_id`. An app has to be deployed before it can be routed to.",
				Optional:            true,
			},
			"bucket_id": schema.StringAttribute{
				MarkdownDescription: "The id of the `cloider_bucket` whose static website to serve. Set this, `app_id` or `cluster_id`. A website takes the whole domain, so `path` and `target_path` stay `/`, and the bucket's `website` has to be on.",
				Optional:            true,
			},
			"loadbalancer": schema.StringAttribute{
				MarkdownDescription: "The LoadBalancer Service in `cluster_id` to point the domain at, as `<namespace>/<service>`. The domain gets an A record to its IP, which follows the IP when the Service is recreated, so it carries any port and protocol and Cloider adds no HTTPS. It takes the whole domain, so `path` and `target_path` stay `/` and the domain can have no other targets. Custom domains only, and the Service needs an IP before the target can be created.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[^/]+/[^/]+$`), "must be <namespace>/<service>"),
					stringvalidator.AlsoRequires(path.MatchRoot("cluster_id")),
					stringvalidator.ConflictsWith(path.MatchRoot("namespace"), path.MatchRoot("service"), path.MatchRoot("port")),
				},
			},
			"loadbalancer_ip": schema.StringAttribute{
				MarkdownDescription: "The IP the domain's A record points at, for a `loadbalancer` target. Empty while the LoadBalancer has none, when the domain has no A record either.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the target was created, RFC 3339.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *targetResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("cluster_id"), path.MatchRoot("app_id"), path.MatchRoot("bucket_id")),
		clusterSource{},
	}
}

// clusterSource checks a cluster_id comes with what to route to in the cluster: a
// LoadBalancer, or a Service's namespace, service and port. Like the attribute
// validators, it leaves an unknown cluster_id alone.
type clusterSource struct{}

func (clusterSource) Description(context.Context) string {
	return "cluster_id needs loadbalancer, or namespace, service and port."
}

func (v clusterSource) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (clusterSource) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var clusterID, loadbalancer, namespace, service types.String
	var port types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("cluster_id"), &clusterID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("loadbalancer"), &loadbalancer)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("namespace"), &namespace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("service"), &service)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("port"), &port)...)
	if resp.Diagnostics.HasError() || clusterID.IsNull() || clusterID.IsUnknown() || !loadbalancer.IsNull() {
		return
	}
	if namespace.IsNull() || service.IsNull() || port.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("cluster_id"), "Invalid Attribute Combination",
			"With cluster_id, set loadbalancer, or namespace, service and port.")
	}
}

// protocolPlan keeps protocol from state when it isn't set, like UseStateForUnknown,
// except that a LoadBalancer target, which carries every protocol, has none, and a
// target switched from one learns its port's protocol on apply. UseStateForUnknown
// would copy the LoadBalancer target's null into the plan.
type protocolPlan struct{}

func (protocolPlan) Description(context.Context) string {
	return "The value in state, null for a LoadBalancer target."
}

func (m protocolPlan) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (protocolPlan) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsUnknown() || req.ConfigValue.IsUnknown() {
		return
	}
	var loadbalancer types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("loadbalancer"), &loadbalancer)...)
	if !loadbalancer.IsNull() {
		resp.PlanValue = types.StringNull()
		return
	}
	if !req.State.Raw.IsNull() && !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

// targetPathDefault plans target_path as path when it isn't set, the way the
// api defaults it, so changing path moves both.
type targetPathDefault struct{}

func (targetPathDefault) Description(context.Context) string {
	return "Defaults to path."
}

func (m targetPathDefault) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (targetPathDefault) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	var p types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("path"), &p)...)
	resp.PlanValue = p
}

func (r *targetResource) form(ctx context.Context, m targetModel) (client.TargetForm, error) {
	form := client.TargetForm{
		DomainName: m.Domain.ValueString(),
		Path:       m.Path.ValueString(),
		TargetPath: m.TargetPath.ValueString(),
	}
	if !m.AppID.IsNull() {
		form.TargetType = "App"
		form.AppId = m.AppID.ValueString()
		return form, nil
	}
	if !m.BucketID.IsNull() {
		form.TargetType = "Bucket"
		form.BucketId = m.BucketID.ValueString()
		return form, nil
	}
	if !m.Loadbalancer.IsNull() {
		namespace, service, _ := strings.Cut(m.Loadbalancer.ValueString(), "/")
		form.TargetType = "LoadBalancer"
		form.ClusterId = m.ClusterID.ValueString()
		form.Namespace = namespace
		form.Service = service
		return form, nil
	}

	// cluster_name is only shown in the dashboard, the api routes by cluster_id
	cluster, err := findIn(ctx, r.client, "/clusters", func(c client.Cluster) string { return c.HashedId }, m.ClusterID.ValueString())
	if err != nil {
		return form, err
	}
	if cluster == nil {
		return form, fmt.Errorf("there is no cluster %s", m.ClusterID.ValueString())
	}

	form.TargetType = "Cluster"
	form.ClusterId = cluster.HashedId
	form.ClusterName = cluster.ClusterName
	form.Namespace = m.Namespace.ValueString()
	form.Service = m.Service.ValueString()
	form.Port = int(m.Port.ValueInt64())
	form.Protocol = "TCP"
	if !m.Protocol.IsNull() && !m.Protocol.IsUnknown() {
		form.Protocol = m.Protocol.ValueString()
	}
	return form, nil
}

func (r *targetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan targetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	form, err := r.form(ctx, plan)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't route "+plan.Domain.ValueString(), err)
		return
	}

	var created client.Created
	if err := r.client.Post(ctx, "/targets", form, &created); err != nil {
		addCreateError(&resp.Diagnostics, "Can't route "+plan.Domain.ValueString()+plan.Path.ValueString(), err)
		return
	}

	plan.ID = types.StringValue(created.HashedId)
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the target back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *targetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state targetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.read(ctx, &state)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read target "+state.ID.ValueString(), err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *targetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state targetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	form, err := r.form(ctx, plan)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't update target "+state.ID.ValueString(), err)
		return
	}

	if err := r.client.Put(ctx, "/targets/"+url.PathEscape(state.ID.ValueString()), form, nil); err != nil {
		addAPIError(&resp.Diagnostics, "Can't update target "+state.ID.ValueString(), err)
		return
	}

	plan.ID = state.ID
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the target back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *targetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state targetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/targets/"+url.PathEscape(state.ID.ValueString()), nil, nil); err != nil && !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't delete target "+state.ID.ValueString(), err)
	}
}

// ImportState takes <domain>/<id>: the api only lists targets by domain
func (r *targetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domain, id, ok := strings.Cut(req.ID, "/")
	if !ok || domain == "" || id == "" {
		resp.Diagnostics.AddError("Invalid target import id", "Import a target as <domain>/<id>, such as example.com/t4k9vq. `cloider target list <domain> -o wide` shows the ids.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), strings.TrimSuffix(strings.ToLower(domain), "."))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// read fills m from the api, false when the target or its domain is gone
func (r *targetResource) read(ctx context.Context, m *targetModel) (bool, error) {
	var targets []client.Target
	err := r.client.Get(ctx, "/targets", url.Values{"domain_name": {m.Domain.ValueString()}}, &targets)
	// The api answers 403 for a domain that isn't the account's, as once it is deleted
	if status := client.StatusOf(err); status == http.StatusForbidden || status == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for _, t := range targets {
		if t.HashedId == m.ID.ValueString() {
			fillTarget(m, t)
			return true, nil
		}
	}
	return false, nil
}

func fillTarget(m *targetModel, t client.Target) {
	m.Domain = types.StringValue(t.DomainName)
	m.Path = types.StringValue(t.Path)
	m.TargetPath = types.StringValue(t.TargetPath)
	m.Protocol = stringOrNull(t.Protocol)
	m.CreatedAt = timeString(t.CreatedAt)

	// An app or bucket target's routing comes from the app or the website
	// service, so only a cluster target has it
	switch t.TargetType {
	case "App":
		m.AppID, m.BucketID, m.Loadbalancer, m.LoadbalancerIP = types.StringValue(t.AppId), types.StringNull(), types.StringNull(), types.StringNull()
		m.ClusterID, m.Namespace, m.Service, m.Port = types.StringNull(), types.StringNull(), types.StringNull(), types.Int64Null()
		return
	case "Bucket":
		m.AppID, m.BucketID, m.Loadbalancer, m.LoadbalancerIP = types.StringNull(), types.StringValue(t.BucketId), types.StringNull(), types.StringNull()
		m.ClusterID, m.Namespace, m.Service, m.Port = types.StringNull(), types.StringNull(), types.StringNull(), types.Int64Null()
		return
	case "LoadBalancer":
		m.AppID, m.BucketID = types.StringNull(), types.StringNull()
		m.ClusterID = types.StringValue(t.ClusterId)
		m.Loadbalancer = types.StringValue(t.Namespace + "/" + t.Service)
		m.LoadbalancerIP = types.StringValue(t.LoadbalancerIp)
		m.Namespace, m.Service, m.Port = types.StringNull(), types.StringNull(), types.Int64Null()
		return
	}
	m.AppID, m.BucketID, m.Loadbalancer, m.LoadbalancerIP = types.StringNull(), types.StringNull(), types.StringNull(), types.StringNull()
	m.ClusterID = types.StringValue(t.ClusterId)
	m.Namespace = types.StringValue(t.Namespace)
	m.Service = types.StringValue(t.Service)
	m.Port = types.Int64Value(int64(t.Port))
}
