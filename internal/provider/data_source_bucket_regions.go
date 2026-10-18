package provider

import (
	"context"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSourceWithConfigure = &bucketRegionsDataSource{}

type bucketRegionsDataSource struct {
	dataSourceBase
}

func NewBucketRegionsDataSource() datasource.DataSource {
	return &bucketRegionsDataSource{}
}

type bucketRegionsModel struct {
	Regions []bucketRegionModel `tfsdk:"regions"`
}

type bucketRegionModel struct {
	Code     string `tfsdk:"code"`
	Name     string `tfsdk:"name"`
	Endpoint string `tfsdk:"endpoint"`
}

func (d *bucketRegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_regions"
}

func (d *bucketRegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The regions a `cloider_bucket` can be created in.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "The regions, the default one first.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"code":     schema.StringAttribute{MarkdownDescription: "The code `cloider_bucket`'s `region` takes, such as `eu-central-2`.", Computed: true},
						"name":     schema.StringAttribute{MarkdownDescription: "The region's name.", Computed: true},
						"endpoint": schema.StringAttribute{MarkdownDescription: "The S3 endpoint of its buckets.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *bucketRegionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var regions []client.BucketRegion
	if err := d.client.Get(ctx, "/buckets/regions", nil, &regions); err != nil {
		addAPIError(&resp.Diagnostics, "Can't read the bucket regions", err)
		return
	}

	model := bucketRegionsModel{Regions: []bucketRegionModel{}}
	for _, r := range regions {
		model.Regions = append(model.Regions, bucketRegionModel{Code: r.Code, Name: r.Name, Endpoint: r.Endpoint})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
