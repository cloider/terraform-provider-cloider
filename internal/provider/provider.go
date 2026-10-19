package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &cloiderProvider{}

type cloiderProvider struct {
	// The release version, "dev" for a local build, "test" in acceptance tests.
	version string
}

type providerModel struct {
	APIURL    types.String `tfsdk:"api_url"`
	Token     types.String `tfsdk:"token"`
	TokenFile types.String `tfsdk:"token_file"`
	Profile   types.String `tfsdk:"profile"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &cloiderProvider{version: version}
	}
}

func (p *cloiderProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloider"
	resp.Version = p.version
}

func (p *cloiderProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Cloider clusters, databases, buckets, container registry projects, domains, DNS records and the targets that route domains to clusters and apps, with an API token.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The api to talk to. Defaults to `CLOIDER_API_URL`, then `" + client.DefaultAPIURL + "`.",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "An API token (`cldr_...`) from Access Control, API Tokens in the dashboard. Defaults to `CLOIDER_TOKEN`. A Read & Write token is needed to apply, a Read token is enough to plan.",
				Optional:            true,
				Sensitive:           true,
			},
			"token_file": schema.StringAttribute{
				MarkdownDescription: "A file holding the API token, used when `token` isn't set. Defaults to `CLOIDER_TOKEN_FILE`.",
				Optional:            true,
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: "The `cloider` CLI profile whose token to use when no other token is set, so `cloider login` is enough on your own machine. Defaults to `CLOIDER_PROFILE`, then the CLI's current profile.",
				Optional:            true,
			},
		},
	}
}

func (p *cloiderProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for name, value := range map[string]types.String{"api_url": config.APIURL, "token": config.Token, "token_file": config.TokenFile, "profile": config.Profile} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Unknown "+name, "The provider's "+name+" has to be known before planning, not come from a resource that isn't created yet.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	session, err := client.Resolve(client.Settings{
		APIURL:    config.APIURL.ValueString(),
		Token:     config.Token.ValueString(),
		TokenFile: config.TokenFile.ValueString(),
		Profile:   config.Profile.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("No Cloider API token", err.Error())
		return
	}

	insecure := os.Getenv("CLOIDER_INSECURE") == "1"
	userAgent := fmt.Sprintf("terraform-provider-cloider/%s (terraform/%s)", p.version, req.TerraformVersion)
	c := client.New(session.APIURL, session.Token, userAgent, insecure)

	// Fails early on a revoked or mistyped token, before any resource is planned
	var whoami client.Whoami
	if err := c.Get(ctx, "/whoami", nil, &whoami); err != nil {
		if client.StatusOf(err) == http.StatusUnauthorized {
			resp.Diagnostics.AddError("Cloider refused the API token", fmt.Sprintf("The token from %s was refused: %s", session.Source, err))
			return
		}
		resp.Diagnostics.AddError("Can't reach the Cloider api", fmt.Sprintf("GET %s/whoami: %s", session.APIURL, err))
		return
	}
	tflog.Debug(ctx, "cloider provider configured", map[string]any{"api_url": session.APIURL, "token_source": session.Source, "role": whoami.Role})

	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *cloiderProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
		NewDatabaseResource,
		NewBucketResource,
		NewBucketKeyResource,
		NewRegistryResource,
		NewRegistryTokenResource,
		NewDomainResource,
		NewDnsRecordResource,
		NewTargetResource,
	}
}

func (p *cloiderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOptionsDataSource,
		NewBucketRegionsDataSource,
		NewClusterDataSource,
	}
}
