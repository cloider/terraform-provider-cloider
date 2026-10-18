package provider

import (
	"context"
	"strings"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &optionsDataSource{}

type optionsDataSource struct {
	dataSourceBase
}

func NewOptionsDataSource() datasource.DataSource {
	return &optionsDataSource{}
}

type optionsModel struct {
	ClusterVersions []string              `tfsdk:"cluster_versions"`
	DatabaseEngines []databaseEngineModel `tfsdk:"database_engines"`
	DatabasePlans   []string              `tfsdk:"database_plans"`
	SubdomainRoots  []string              `tfsdk:"subdomain_roots"`
	Nameservers     []string              `tfsdk:"nameservers"`
}

type databaseEngineModel struct {
	Engine   string   `tfsdk:"engine"`
	Versions []string `tfsdk:"versions"`
	Ssl      bool     `tfsdk:"ssl"`
}

func (d *optionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_options"
}

func (d *optionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	stringList := func(description string) schema.ListAttribute {
		return schema.ListAttribute{MarkdownDescription: description, ElementType: types.StringType, Computed: true}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "What can be created at the moment: Kubernetes versions, database engines and plans, subdomain roots, and the nameservers a custom domain points at.",
		Attributes: map[string]schema.Attribute{
			"cluster_versions": stringList("The Kubernetes versions a `cloider_cluster` can run, newest first."),
			"database_engines": schema.ListNestedAttribute{
				MarkdownDescription: "The engines a `cloider_database` can run.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"engine":   schema.StringAttribute{MarkdownDescription: "The engine, as `cloider_database` takes it.", Computed: true},
						"versions": stringList("Its versions, newest first."),
						"ssl":      schema.BoolAttribute{MarkdownDescription: "Whether it supports SSL.", Computed: true},
					},
				},
			},
			"database_plans":  stringList("The plans a `cloider_database` can have, smallest first."),
			"subdomain_roots": stringList("The roots a free subdomain `cloider_domain` goes under, such as `cloider.app`."),
			"nameservers":     stringList("The nameservers to point a custom `cloider_domain` at, at its registrar."),
		},
	}
}

func (d *optionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var options client.Options
	if err := d.client.Get(ctx, "/options", nil, &options); err != nil {
		addAPIError(&resp.Diagnostics, "Can't read the options", err)
		return
	}

	// Lower case, as the resources take engines and plans
	model := optionsModel{
		ClusterVersions: nonNil(options.ClusterVersions),
		DatabaseEngines: []databaseEngineModel{},
		DatabasePlans:   []string{},
		SubdomainRoots:  nonNil(options.SubdomainRoots),
		Nameservers:     nonNil(options.Nameservers),
	}
	for _, e := range options.DatabaseEngines {
		model.DatabaseEngines = append(model.DatabaseEngines, databaseEngineModel{Engine: strings.ToLower(e.Service), Versions: nonNil(e.Versions), Ssl: e.Ssl})
	}
	for _, p := range options.DatabasePlans {
		model.DatabasePlans = append(model.DatabasePlans, strings.ToLower(p))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
