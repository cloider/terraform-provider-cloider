package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSourceWithConfigure        = &clusterDataSource{}
	_ datasource.DataSourceWithConfigValidators = &clusterDataSource{}
)

type clusterDataSource struct {
	dataSourceBase
}

func NewClusterDataSource() datasource.DataSource {
	return &clusterDataSource{}
}

type clusterDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Version    types.String `tfsdk:"version"`
	Label      types.String `tfsdk:"label"`
	Status     types.String `tfsdk:"status"`
	CreatedAt  types.String `tfsdk:"created_at"`
	KubeConfig types.Object `tfsdk:"kube_config"`
}

func (d *clusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *clusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	kubeString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true, Sensitive: true}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "An existing Kubernetes cluster, found by `id` or `name`, with what the `kubernetes` and `helm` providers need to deploy into it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The cluster's id. Set this or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The cluster's name. Set this or `id`. Names don't have to be unique, so a name several clusters have is refused, use the `id` then.",
				Optional:            true,
				Computed:            true,
			},
			"version":    schema.StringAttribute{MarkdownDescription: "The Kubernetes version.", Computed: true},
			"label":      schema.StringAttribute{MarkdownDescription: "The name the dashboard shows.", Computed: true},
			"status":     schema.StringAttribute{MarkdownDescription: "`Creating`, `Starting`, `Ready`, `Pausing`, `Paused` or `Terminating`.", Computed: true},
			"created_at": schema.StringAttribute{MarkdownDescription: "When the cluster was created, RFC 3339.", Computed: true},
			"kube_config": schema.SingleNestedAttribute{
				MarkdownDescription: "How to reach the cluster's Kubernetes api, for the `kubernetes` and `helm` providers. Null until the cluster is ready.",
				Computed:            true,
				Sensitive:           true,
				Attributes: map[string]schema.Attribute{
					"raw":                    kubeString("The whole kubeconfig file."),
					"host":                   kubeString("The Kubernetes api's url."),
					"cluster_ca_certificate": kubeString("The PEM CA certificate the api's certificate is signed by."),
					"client_certificate":     kubeString("The PEM client certificate to authenticate with."),
					"client_key":             kubeString("The PEM key of `client_certificate`."),
					"token":                  kubeString("The bearer token to authenticate with, when the kubeconfig uses one."),
				},
			},
		},
	}
}

func (d *clusterDataSource) ConfigValidators(context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config clusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var clusters []client.Cluster
	if err := d.client.Get(ctx, "/clusters", nil, &clusters); err != nil {
		addAPIError(&resp.Diagnostics, "Can't list the clusters", err)
		return
	}

	var found []client.Cluster
	for _, c := range clusters {
		if c.Status == "Terminated" {
			continue
		}
		if (!config.ID.IsNull() && c.HashedId == config.ID.ValueString()) || (!config.Name.IsNull() && c.ClusterName == config.Name.ValueString()) {
			found = append(found, c)
		}
	}
	switch {
	case len(found) == 0 && !config.ID.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("id"), "No such cluster", fmt.Sprintf("There is no cluster %q, see `cloider cluster list -o wide`.", config.ID.ValueString()))
		return
	case len(found) == 0:
		resp.Diagnostics.AddAttributeError(path.Root("name"), "No such cluster", fmt.Sprintf("There is no cluster named %q, see `cloider cluster list`.", config.Name.ValueString()))
		return
	case len(found) > 1:
		var ids []string
		for _, c := range found {
			ids = append(ids, fmt.Sprintf("%s (%s)", c.HashedId, c.ClusterLabel))
		}
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Several clusters have this name", fmt.Sprintf("Clusters %s are all named %q, give the id of the one to use.", strings.Join(ids, ", "), config.Name.ValueString()))
		return
	}

	c := found[0]
	state := clusterDataSourceModel{
		ID:         types.StringValue(c.HashedId),
		Name:       types.StringValue(c.ClusterName),
		Version:    types.StringValue(c.ClusterVersion),
		Label:      types.StringValue(c.ClusterLabel),
		Status:     types.StringValue(c.Status),
		CreatedAt:  timeString(c.CreatedAt),
		KubeConfig: types.ObjectNull(kubeConfigAttrTypes),
	}

	kubeconfig, err := d.client.GetText(ctx, "/clusters/"+c.HashedId+"/kubeconfig", nil)
	if err != nil && !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't read cluster "+c.HashedId+"'s kubeconfig", err)
		return
	}
	if err == nil {
		state.KubeConfig = kubeConfigValue(kubeconfig)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
