// Package provider implements the Zinn Digital Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/Zinn-Digital/terraform-provider-zinn/internal/zinnapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &zinnProvider{}

type zinnProvider struct{ version string }

type providerModel struct {
	APIURL types.String `tfsdk:"api_url"`
	APIKey types.String `tfsdk:"api_key"`
	Org    types.String `tfsdk:"org"`
}

// New returns the provider factory for the given release version.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &zinnProvider{version: version} }
}

func (p *zinnProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zinn"
	resp.Version = p.version
}

func (p *zinnProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Zinn Digital hosting through the public API. Catalogue data sources need no credential; " +
			"everything else uses a per-organisation API key.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "API base URL. Defaults to `ZINN_API_URL`, then `https://api.zinndigital.com`.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Per-organisation API key. Defaults to `ZINN_API_KEY`.",
			},
			"org": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Organisation id to act on (sent as `X-Zinn-Org`). Defaults to `ZINN_ORG`; usually unnecessary because a key belongs to one organisation.",
			},
		},
	}
}

func pick(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}

func (p *zinnProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := zinnapi.New(pick(cfg.APIURL, "ZINN_API_URL"), pick(cfg.APIKey, "ZINN_API_KEY"), pick(cfg.Org, "ZINN_ORG"),
		"terraform-provider-zinn/"+p.version)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *zinnProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{newDNSRecordResource}
}

func (p *zinnProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newRegionsDataSource, newPlansDataSource, newSitesDataSource, newDomainsDataSource}
}
