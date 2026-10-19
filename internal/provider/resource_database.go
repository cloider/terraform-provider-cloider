package provider

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
	_ resource.ResourceWithModifyPlan  = &databaseResource{}
)

// databaseEngines is the CLI's --engine names for the api's services.
var databaseEngines = map[string]string{
	"postgresql": "PostgreSQL",
	"mysql":      "MySQL",
	"mongodb":    "MongoDB",
	"redis":      "Redis",
}

// databasePlans is smallest first: a plan can only be resized up.
var databasePlans = []string{"Hobby", "Starter", "Basic", "Standard", "Plus", "Pro"}

func planIndex(plan string) int {
	return slices.IndexFunc(databasePlans, func(p string) bool { return strings.EqualFold(p, plan) })
}

type databaseResource struct {
	resourceBase
}

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

type databaseModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Engine         types.String   `tfsdk:"engine"`
	Version        types.String   `tfsdk:"version"`
	User           types.String   `tfsdk:"user"`
	Plan           types.String   `tfsdk:"plan"`
	Label          types.String   `tfsdk:"label"`
	Ssl            types.Bool     `tfsdk:"ssl"`
	TrustedSources types.Set      `tfsdk:"trusted_sources"`
	Paused         types.Bool     `tfsdk:"paused"`
	Status         types.String   `tfsdk:"status"`
	Host           types.String   `tfsdk:"host"`
	PrivateHost    types.String   `tfsdk:"private_host"`
	Port           types.Int64    `tfsdk:"port"`
	Password       types.String   `tfsdk:"password"`
	CACertificate  types.String   `tfsdk:"ca_certificate"`
	CreatedAt      types.String   `tfsdk:"created_at"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *databaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	lowerPlans := make([]string, len(databasePlans))
	for i, p := range databasePlans {
		lowerPlans[i] = strings.ToLower(p)
	}
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "A managed database: PostgreSQL, MySQL, MongoDB or Redis. Creating one waits until it is `Ready`. `name`, `engine`, `version` and `user` can't change, and a plan can only be resized up, so changing those replaces the database and its data.",
		Attributes: map[string]schema.Attribute{
			"id": computedString("The database's id."),
			"name": schema.StringAttribute{
				MarkdownDescription: "The database's name: lower case letters, digits and `-`, starting with a letter.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"engine": schema.StringAttribute{
				MarkdownDescription: "`postgresql`, `mysql`, `mongodb` or `redis`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf("postgresql", "mysql", "mongodb", "redis")},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The engine version, from `cloider_options`' `database_engines`. Defaults to the newest offered.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "The user to connect as: lower case letters, digits and `_`. Defaults to `<name>_user`, with `-` as `_`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"plan": schema.StringAttribute{
				MarkdownDescription: "`hobby`, `starter`, `basic`, `standard`, `plus` or `pro`, see `cloider pricing`. Moving to a larger plan resizes the database in place, with a restart. A smaller one replaces it. Defaults to `hobby`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("hobby"),
				Validators:          []validator.String{stringvalidator.OneOf(lowerPlans...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(
					func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						resp.RequiresReplace = planIndex(req.PlanValue.ValueString()) < planIndex(req.StateValue.ValueString())
					},
					"A smaller plan replaces the database, plans can only be resized up.",
					"A smaller plan replaces the database, plans can only be resized up.",
				)},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "The name the dashboard shows: lower case letters, digits and `-`, starting with a letter. Defaults to `name`, with a `-1`, `-2`... suffix when another database has it. It has to be unique among your databases.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ssl": schema.BoolAttribute{
				MarkdownDescription: "Whether connections must use TLS. Changing it restarts the database. Defaults to on, except for Redis, where it makes the server TLS only.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"trusted_sources": schema.SetAttribute{
				MarkdownDescription: "Who may connect: `*` for anywhere, an IPv4 CIDR such as `203.0.113.0/24` (`/32` for one address), `cluster:<id>` for one of your clusters or `app:<id>` for one of your apps. Defaults to `[\"*\"]`, as the dashboard does.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{types.StringValue("*")})),
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(trustedSourceValidator{}),
					anywhereAloneValidator{},
				},
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Whether the database is paused. A paused database keeps its data and can't be connected to. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "`Creating`, `Starting`, `Ready`, `Upgrading`, `Pausing`, `Paused` or `Terminating`.",
				Computed:            true,
			},
			"host":         computedString("The host to connect to from anywhere the trusted sources allow."),
			"private_host": computedString("The host to connect to from your clusters and apps, inside Cloider's network."),
			"port": schema.Int64Attribute{
				MarkdownDescription: "The port to connect to.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of `user`.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ca_certificate": schema.StringAttribute{
				MarkdownDescription: "The PEM CA certificate that signs the database's TLS certificate, when `ssl` is on.",
				Computed:            true,
				Sensitive:           true,
			},
			"created_at": computedString("When the database was created, RFC 3339."),
			"timeouts":   timeouts.Attributes(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
		},
	}
}

// ModifyPlan refuses at plan time what a paused database can't do, rather than
// failing halfway through an apply: a resize needs it Ready, and changing SSL
// restarts it, which would un-pause it.
func (r *databaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || len(resp.RequiresReplace) > 0 {
		return
	}
	var plan, state databaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.Paused.ValueBool() || !plan.Paused.ValueBool() {
		return
	}
	if !plan.Plan.IsUnknown() && !plan.Plan.Equal(state.Plan) {
		resp.Diagnostics.AddAttributeError(path.Root("plan"), "Can't resize a paused database", "Resume it with paused = false to resize it, then pause it again.")
	}
	if !plan.Ssl.IsUnknown() && !plan.Ssl.Equal(state.Ssl) {
		resp.Diagnostics.AddAttributeError(path.Root("ssl"), "Can't change SSL on a paused database", "Changing SSL restarts the database. Resume it with paused = false first.")
	}
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseModel
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

	// The CLI's defaults, which come from what the api offers
	var options client.Options
	if err := r.client.Get(ctx, "/options", nil, &options); err != nil {
		addAPIError(&resp.Diagnostics, "Can't read the database engines offered", err)
		return
	}
	service := databaseEngines[plan.Engine.ValueString()]
	i := slices.IndexFunc(options.DatabaseEngines, func(e client.DatabaseEngineOption) bool { return e.Service == service })
	if i < 0 {
		resp.Diagnostics.AddAttributeError(path.Root("engine"), "Engine not offered", service+" isn't offered at the moment.")
		return
	}
	engine := options.DatabaseEngines[i]

	version := plan.Version.ValueString()
	if version == "" {
		if len(engine.Versions) == 0 {
			resp.Diagnostics.AddAttributeError(path.Root("version"), "No version offered", "The api offers no "+service+" version at the moment.")
			return
		}
		version = engine.Versions[0]
	}
	user := plan.User.ValueString()
	if user == "" {
		user = strings.ReplaceAll(plan.Name.ValueString(), "-", "_") + "_user"
	}
	ssl := engine.Ssl && service != "Redis"
	if !plan.Ssl.IsUnknown() && !plan.Ssl.IsNull() {
		ssl = plan.Ssl.ValueBool()
	}
	sources, d := r.trustedSources(ctx, plan.TrustedSources)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := client.DatabaseForm{
		DatabaseName:    plan.Name.ValueString(),
		DatabaseUser:    user,
		Service:         service,
		DatabaseVersion: version,
		DatabasePlan:    databasePlans[planIndex(plan.Plan.ValueString())],
		TrustedSources:  sources,
		Ssl:             ssl,
	}
	var created client.Created
	if err := r.client.Post(ctx, "/databases", form, &created); err != nil {
		addCreateError(&resp.Diagnostics, "Can't create the database", err)
		return
	}

	// In state before waiting, so a failed wait leaves it tainted, not orphaned
	state := plan
	state.ID = types.StringValue(created.HashedId)
	state.Version = types.StringValue(version)
	state.User = types.StringValue(user)
	state.Ssl = types.BoolValue(ssl)
	state.Paused = types.BoolValue(false)
	state.Label = types.StringNull()
	state.Status = types.StringValue("Creating")
	state.Host, state.PrivateHost, state.Port = types.StringNull(), types.StringNull(), types.Int64Null()
	state.Password, state.CACertificate, state.CreatedAt = types.StringNull(), types.StringNull(), types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	basePath := "/databases/" + created.HashedId
	what := "database " + plan.Name.ValueString()
	if _, err := r.client.WaitStatus(ctx, basePath+"/status", "Ready"); err != nil {
		resp.Diagnostics.AddError(what+" didn't become ready", err.Error())
		return
	}
	if label := plan.Label.ValueString(); label != "" {
		if err := r.client.Put(ctx, basePath+"/label", map[string]string{"database_label": label}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't set the label of "+what, err)
			return
		}
	}
	if plan.Paused.ValueBool() {
		if !setPaused(ctx, r.client, basePath, true, what, &resp.Diagnostics) {
			return
		}
	}

	found, err := r.read(ctx, &state)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the database back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.read(ctx, &state)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read database "+state.ID.ValueString(), err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state databaseModel
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

	basePath := "/databases/" + state.ID.ValueString()
	what := "database " + state.Name.ValueString()

	// Resume first and pause last, the changes in between need it running
	if state.Paused.ValueBool() && !plan.Paused.ValueBool() {
		if !setPaused(ctx, r.client, basePath, false, what, &resp.Diagnostics) {
			return
		}
	}
	if !plan.Label.IsUnknown() && !plan.Label.Equal(state.Label) {
		if err := r.client.Put(ctx, basePath+"/label", map[string]string{"database_label": plan.Label.ValueString()}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change the label of "+what, err)
			return
		}
	}
	if !plan.TrustedSources.Equal(state.TrustedSources) {
		sources, d := r.trustedSources(ctx, plan.TrustedSources)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.Put(ctx, basePath+"/trusted-sources", map[string]any{"trusted_sources": sources}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change who may connect to "+what, err)
			return
		}
	}
	if !plan.Ssl.IsUnknown() && !plan.Ssl.Equal(state.Ssl) {
		if err := r.client.Put(ctx, basePath+"/ssl", map[string]bool{"ssl": plan.Ssl.ValueBool()}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't change SSL of "+what, err)
			return
		}
		if _, err := r.client.WaitStatus(ctx, basePath+"/status", "Ready"); err != nil {
			resp.Diagnostics.AddError(what+" didn't restart with the new SSL setting", err.Error())
			return
		}
	}
	if !plan.Plan.Equal(state.Plan) {
		target := databasePlans[planIndex(plan.Plan.ValueString())]
		if err := r.client.Put(ctx, basePath+"/plan", map[string]string{"database_plan": target}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "Can't resize "+what, err)
			return
		}
		if _, err := r.client.WaitStatus(ctx, basePath+"/status", "Ready"); err != nil {
			resp.Diagnostics.AddError(what+" didn't finish resizing", err.Error())
			return
		}
		// A resize that fails goes back to Ready on the old plan
		var db client.Database
		if err := r.client.Get(ctx, basePath, nil, &db); err != nil {
			addAPIError(&resp.Diagnostics, "Can't read "+what+" back", err)
			return
		}
		if db.DatabasePlan != target {
			resp.Diagnostics.AddError("Resizing "+what+" failed", fmt.Sprintf("It is still on %s. Apply again to retry, or contact support if it keeps failing.", db.DatabasePlan))
			return
		}
	}
	if !state.Paused.ValueBool() && plan.Paused.ValueBool() {
		if !setPaused(ctx, r.client, basePath, true, what, &resp.Diagnostics) {
			return
		}
	}

	plan.ID = state.ID
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Can't read the database back", errOrGone(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseModel
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

	deleteAndWait(ctx, r.client, "/databases/"+state.ID.ValueString(), "database "+state.Name.ValueString(), &resp.Diagnostics)
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read fills m from the api, false when the database is gone.
func (r *databaseResource) read(ctx context.Context, m *databaseModel) (bool, error) {
	id := m.ID.ValueString()
	var db client.Database
	err := r.client.Get(ctx, "/databases/"+id, nil, &db)
	if client.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if db.Status == "Terminated" {
		return false, nil
	}

	sources := make([]string, 0, len(db.TrustedSources))
	for _, s := range db.TrustedSources {
		sources = append(sources, trustedSourceString(s))
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, sources)
	if diags.HasError() {
		return false, fmt.Errorf("can't read the trusted sources: %v", diags)
	}

	m.Name = types.StringValue(db.DatabaseName)
	m.Engine = types.StringValue(strings.ToLower(db.Service))
	m.Version = types.StringValue(db.DatabaseVersion)
	m.User = types.StringValue(db.DatabaseUser)
	m.Plan = types.StringValue(strings.ToLower(db.DatabasePlan))
	m.Label = types.StringValue(db.DatabaseLabel)
	m.Ssl = types.BoolValue(db.Ssl)
	m.TrustedSources = set
	m.Paused = types.BoolValue(paused(db.Status))
	m.Status = types.StringValue(db.Status)
	m.Host = stringOrNull(db.Host)
	m.PrivateHost = stringOrNull(db.PrivateHost)
	m.Port = types.Int64Null()
	if port, err := strconv.ParseInt(db.DatabasePort, 10, 64); err == nil {
		m.Port = types.Int64Value(port)
	}
	m.CreatedAt = timeString(db.CreatedAt)

	// Each of these reads the database's secrets on its host cluster, and neither
	// changes on its own, so only what state doesn't have yet is fetched
	query := url.Values{"id": {id}}
	if m.Password.IsNull() || m.Password.IsUnknown() {
		password, err := r.client.GetText(ctx, "/database/password", query)
		if err != nil {
			return false, fmt.Errorf("can't read the password: %w", err)
		}
		m.Password = stringOrNull(password)
	}

	if !db.Ssl {
		m.CACertificate = types.StringNull()
	} else if m.CACertificate.IsNull() || m.CACertificate.IsUnknown() {
		cert, err := r.client.GetText(ctx, "/database/cert", query)
		if err != nil && !client.IsNotFound(err) {
			return false, fmt.Errorf("can't read the CA certificate: %w", err)
		}
		m.CACertificate = stringOrNull(cert)
	}
	return true, nil
}

// trustedSources turns the attribute into what the api stores. A cluster or app
// source carries its name, which the database's firewall matches it by.
func (r *databaseResource) trustedSources(ctx context.Context, set types.Set) ([]client.TrustedSource, diag.Diagnostics) {
	values, diags := stringsOf(ctx, set)
	if diags.HasError() {
		return nil, diags
	}

	var clusters []client.Cluster
	sources := make([]client.TrustedSource, 0, len(values))
	for _, value := range values {
		kind, id, _ := strings.Cut(value, ":")
		switch {
		case value == "*":
			sources = append(sources, client.TrustedSource{Name: "Allow Everything", Code: "*", Type: "Anywhere"})
		case kind == "cluster":
			if clusters == nil {
				if err := r.client.Get(ctx, "/clusters", nil, &clusters); err != nil {
					diags.AddAttributeError(path.Root("trusted_sources"), "Can't list clusters", err.Error())
					return nil, diags
				}
			}
			i := slices.IndexFunc(clusters, func(c client.Cluster) bool { return c.HashedId == id })
			if i < 0 {
				diags.AddAttributeError(path.Root("trusted_sources"), "Unknown cluster", "No cluster has the id "+id+".")
				return nil, diags
			}
			sources = append(sources, client.TrustedSource{Name: clusters[i].ClusterName, Code: id, Type: "Cluster"})
		case kind == "app":
			var app client.App
			if err := r.client.Get(ctx, "/apps/"+id, nil, &app); err != nil {
				diags.AddAttributeError(path.Root("trusted_sources"), "Unknown app", fmt.Sprintf("Can't find the app with the id %s: %s", id, err))
				return nil, diags
			}
			sources = append(sources, client.TrustedSource{Name: app.AppName, Code: id, Type: "App"})
		default:
			sources = append(sources, client.TrustedSource{Name: value, Code: value, Type: "Custom"})
		}
	}
	return sources, diags
}

func trustedSourceString(s client.TrustedSource) string {
	switch s.Type {
	case "Anywhere":
		return "*"
	case "Cluster":
		return "cluster:" + s.Code
	case "App":
		return "app:" + s.Code
	}
	return s.Code
}
