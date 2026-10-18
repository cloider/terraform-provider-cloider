package provider

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.ResourceWithConfigure   = &dnsRecordResource{}
	_ resource.ResourceWithImportState = &dnsRecordResource{}
)

var dnsRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "CAA", "HTTPS", "SPF", "SRV", "PTR"}

type dnsRecordResource struct {
	resourceBase
}

func NewDnsRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

type dnsRecordModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
	TTL    types.Int64  `tfsdk:"ttl"`
}

// The api trims each value and drops empty ones, so only single trimmed lines
// read back as written.
var dnsRecordValue = regexp.MustCompile(`^\S(?:[^\n]*\S)?$`)

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A DNS record set of a custom `cloider_domain`: every value of one type at one name. The api keeps one record per type and name, so list all of a name's values in one resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The record's id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The custom domain the record belongs to. Subdomains under Cloider's roots don't take records. Changing it replaces the record.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "`" + strings.Join(dnsRecordTypes, "`, `") + "`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(dnsRecordTypes...)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name under the domain, such as `www`. Defaults to `\"\"`, the domain itself.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"values": schema.ListAttribute{
				MarkdownDescription: "The record's values, such as IP addresses for `A` or `10 mail.example.com.` for `MX`. A `CNAME` takes one.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(dnsRecordValue, "must be one line, without leading or trailing spaces")),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Seconds resolvers may cache the record, 60 to 172800.",
				Required:            true,
				Validators:          []validator.Int64{int64validator.Between(60, 172800)},
			},
		},
	}
}

func (r *dnsRecordResource) form(ctx context.Context, m dnsRecordModel) (client.DnsRecordForm, diag.Diagnostics) {
	var values []string
	diags := m.Values.ElementsAs(ctx, &values, false)
	return client.DnsRecordForm{
		RecordType:  m.Type.ValueString(),
		RecordName:  m.Name.ValueString(),
		RecordValue: strings.Join(values, "\n"),
		Ttl:         int(m.TTL.ValueInt64()),
	}, diags
}

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	form, diags := r.form(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var record client.DnsRecord
	if err := r.client.Post(ctx, "/domains/"+url.PathEscape(plan.Domain.ValueString())+"/dns-records", form, &record); err != nil {
		addCreateError(&resp.Diagnostics, "Can't add the DNS record", err)
		return
	}
	resp.Diagnostics.Append(fillDnsRecord(ctx, &plan, record)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var record client.DnsRecord
	err := r.client.Get(ctx, "/dns-records/"+url.PathEscape(state.ID.ValueString()), nil, &record)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Can't read DNS record "+state.ID.ValueString(), err)
		return
	}
	resp.Diagnostics.Append(fillDnsRecord(ctx, &state, record)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dnsRecordModel
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

	var record client.DnsRecord
	if err := r.client.Put(ctx, "/dns-records/"+url.PathEscape(state.ID.ValueString()), form, &record); err != nil {
		addAPIError(&resp.Diagnostics, "Can't update DNS record "+state.ID.ValueString(), err)
		return
	}
	resp.Diagnostics.Append(fillDnsRecord(ctx, &plan, record)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/dns-records/"+url.PathEscape(state.ID.ValueString()), nil, nil); err != nil && !client.IsNotFound(err) {
		addAPIError(&resp.Diagnostics, "Can't delete DNS record "+state.ID.ValueString(), err)
	}
}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.ParseInt(req.ID, 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid DNS record id", "A DNS record's id is a number, see `cloider dns list <domain>`.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func fillDnsRecord(ctx context.Context, m *dnsRecordModel, record client.DnsRecord) diag.Diagnostics {
	values, diags := types.ListValueFrom(ctx, types.StringType, strings.Split(record.RecordValue, "\n"))
	m.ID = types.StringValue(strconv.FormatInt(record.Id, 10))
	m.Domain = types.StringValue(record.DomainName)
	m.Type = types.StringValue(record.RecordType)
	m.Name = types.StringValue(record.RecordName)
	m.Values = values
	m.TTL = types.Int64Value(int64(record.Ttl))
	return diags
}
