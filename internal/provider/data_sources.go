package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Zinn-Digital/terraform-provider-zinn/internal/zinnapi"
)

// base carries the configured client for every data source.
type base struct{ client *zinnapi.Client }

func (b *base) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	b.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func str(d string) schema.StringAttribute {
	return schema.StringAttribute{Computed: true, MarkdownDescription: d}
}

// ---- zinn_regions -------------------------------------------------------------------------------

type regionsDataSource struct{ base }

func newRegionsDataSource() datasource.DataSource { return &regionsDataSource{} }

type regionModel struct {
	Slug    types.String `tfsdk:"slug"`
	City    types.String `tfsdk:"city"`
	Country types.String `tfsdk:"country"`
}

type regionsModel struct {
	Regions []regionModel `tfsdk:"regions"`
}

func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The hosting regions a site can be placed in. Public: no API key needed.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"slug":    str("Region identifier, used as `region` when creating a site."),
					"city":    str("City the region is in."),
					"country": str("ISO 3166-1 alpha-2 country code."),
				}},
			},
		},
	}
}

func (d *regionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	regions, err := d.client.Regions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Listing regions failed", err.Error())
		return
	}
	var state regionsModel
	state.Regions = []regionModel{}
	for _, r := range regions {
		state.Regions = append(state.Regions, regionModel{
			Slug: types.StringValue(r.Slug), City: types.StringValue(r.City), Country: types.StringValue(r.Country),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- zinn_plans ---------------------------------------------------------------------------------

type plansDataSource struct{ base }

func newPlansDataSource() datasource.DataSource { return &plansDataSource{} }

type planModel struct {
	ID            types.String `tfsdk:"id"`
	Code          types.String `tfsdk:"code"`
	Name          types.String `tfsdk:"name"`
	ProductLine   types.String `tfsdk:"product_line"`
	PlanVersionID types.String `tfsdk:"plan_version_id"`
	Orderable     types.Bool   `tfsdk:"orderable"`
	Recommended   types.Bool   `tfsdk:"recommended"`
	SortOrder     types.Int64  `tfsdk:"sort_order"`
}

type plansModel struct {
	ProductLine types.String `tfsdk:"product_line"`
	Plans       []planModel  `tfsdk:"plans"`
}

func (d *plansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plans"
}

func (d *plansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The public hosting plan catalogue. Public: no API key needed.",
		Attributes: map[string]schema.Attribute{
			"product_line": schema.StringAttribute{Optional: true, MarkdownDescription: "Only plans of this product line."},
			"plans": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":              str("The plan's stable id."),
					"code":            str("The plan code, e.g. `agency_start`."),
					"name":            str("Display name."),
					"product_line":    str("Product line the plan belongs to."),
					"plan_version_id": str("The purchasable version id of the plan."),
					"orderable":       schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the plan can be ordered now."},
					"recommended":     schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the plan is the recommended one in its line."},
					"sort_order":      schema.Int64Attribute{Computed: true, MarkdownDescription: "Display order."},
				}},
			},
		},
	}
}

func (d *plansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state plansModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plans, err := d.client.Plans(ctx, state.ProductLine.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Listing plans failed", err.Error())
		return
	}
	state.Plans = []planModel{}
	for _, p := range plans {
		state.Plans = append(state.Plans, planModel{
			ID: types.StringValue(p.ID), Code: types.StringValue(p.Code), Name: types.StringValue(p.Name),
			ProductLine: types.StringValue(p.ProductLine), PlanVersionID: types.StringValue(p.PlanVersionID),
			Orderable: boolPtr(p.Orderable), Recommended: types.BoolValue(p.Recommended), SortOrder: types.Int64Value(p.SortOrder),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- zinn_sites ---------------------------------------------------------------------------------

type sitesDataSource struct{ base }

func newSitesDataSource() datasource.DataSource { return &sitesDataSource{} }

type siteModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	PrimaryDomain types.String `tfsdk:"primary_domain"`
	Status        types.String `tfsdk:"status"`
	ProductLine   types.String `tfsdk:"product_line"`
	PlanCode      types.String `tfsdk:"plan_code"`
	Region        types.String `tfsdk:"region"`
	StackType     types.String `tfsdk:"stack_type"`
	PHPVersion    types.String `tfsdk:"php_version"`
}

type sitesModel struct {
	Sites []siteModel `tfsdk:"sites"`
}

func (d *sitesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sites"
}

func (d *sitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Every site in the API key's organisation.",
		Attributes: map[string]schema.Attribute{
			"sites": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id": str("Site id."), "name": str("Site name."), "primary_domain": str("Primary domain."),
					"status": str("Lifecycle status."), "product_line": str("Product line."), "plan_code": str("Plan code."),
					"region": str("Region slug."), "stack_type": str("Stack type."), "php_version": str("PHP version, when PHP."),
				}},
			},
		},
	}
}

func (d *sitesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !needsKey(d.client, &resp.Diagnostics) {
		return
	}
	sites, err := d.client.Sites(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Listing sites failed", err.Error())
		return
	}
	state := sitesModel{Sites: []siteModel{}}
	for _, s := range sites {
		state.Sites = append(state.Sites, siteModel{
			ID: types.StringValue(s.ID), Name: strPtr(s.Name), PrimaryDomain: types.StringValue(s.PrimaryDomain),
			Status: types.StringValue(s.Status), ProductLine: types.StringValue(s.ProductLine), PlanCode: strPtr(s.PlanCode),
			Region: strPtr(s.Region), StackType: types.StringValue(s.StackType), PHPVersion: strPtr(s.PhpVersion),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- zinn_domains -------------------------------------------------------------------------------

type domainsDataSource struct{ base }

func newDomainsDataSource() datasource.DataSource { return &domainsDataSource{} }

type domainModel struct {
	ID            types.String `tfsdk:"id"`
	FQDN          types.String `tfsdk:"fqdn"`
	Status        types.String `tfsdk:"status"`
	OwnershipMode types.String `tfsdk:"ownership_mode"`
	AutoRenew     types.Bool   `tfsdk:"auto_renew"`
	Nameservers   []string     `tfsdk:"nameservers"`
}

type domainsModel struct {
	Domains []domainModel `tfsdk:"domains"`
}

func (d *domainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *domainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Every domain in the API key's organisation. Use a domain's `id` as `domain_id` on `zinn_dns_record`.",
		Attributes: map[string]schema.Attribute{
			"domains": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id": str("Domain id."), "fqdn": str("Fully qualified domain name."), "status": str("Status."),
					"ownership_mode": str("How the domain is held."),
					"auto_renew":     schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether it renews automatically."},
					"nameservers":    schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Nameservers."},
				}},
			},
		},
	}
}

func (d *domainsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !needsKey(d.client, &resp.Diagnostics) {
		return
	}
	domains, err := d.client.Domains(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Listing domains failed", err.Error())
		return
	}
	state := domainsModel{Domains: []domainModel{}}
	for _, x := range domains {
		ns := x.Nameservers
		if ns == nil {
			ns = []string{}
		}
		state.Domains = append(state.Domains, domainModel{
			ID: types.StringValue(x.ID), FQDN: types.StringValue(x.Fqdn), Status: types.StringValue(x.Status),
			OwnershipMode: types.StringValue(x.OwnershipMode), AutoRenew: types.BoolValue(x.AutoRenew), Nameservers: ns,
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
