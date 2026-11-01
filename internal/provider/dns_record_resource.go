package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Zinn-Digital/terraform-provider-zinn/internal/zinnapi"
)

var (
	_ resource.Resource                = &dnsRecordResource{}
	_ resource.ResourceWithImportState = &dnsRecordResource{}
)

type dnsRecordResource struct{ client *zinnapi.Client }

func newDNSRecordResource() resource.Resource { return &dnsRecordResource{} }

type dnsRecordModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.String `tfsdk:"domain_id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Value    types.String `tfsdk:"value"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
	Proxied  types.Bool   `tfsdk:"proxied"`
	Owner    types.String `tfsdk:"owner"`
}

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A DNS record on a domain hosted in your organisation's DNS. Import with `<domain_id>/<record_id>`.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, PlanModifiers: keep, MarkdownDescription: "Record id."},
			"domain_id": schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "Id of the domain (see `zinn_domains`)."},
			"name":      schema.StringAttribute{Required: true, MarkdownDescription: "Record name, e.g. `www` or `@`."},
			"type":      schema.StringAttribute{Required: true, MarkdownDescription: "Record type, e.g. `A`, `CNAME`, `TXT`, `MX`."},
			"value":     schema.StringAttribute{Required: true, MarkdownDescription: "Record value."},
			"ttl":       schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "TTL in seconds; the API default when unset."},
			"priority":  schema.Int64Attribute{Optional: true, MarkdownDescription: "Priority, for MX and SRV records."},
			"proxied":   schema.BoolAttribute{Optional: true, MarkdownDescription: "Serve through the CDN proxy, where the zone supports it."},
			"owner":     schema.StringAttribute{Computed: true, PlanModifiers: keep, MarkdownDescription: "Who manages the record (you or the platform)."},
		},
	}
}

func (r *dnsRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func toWrite(m dnsRecordModel) zinnapi.DnsRecordWrite {
	w := zinnapi.DnsRecordWrite{Name: m.Name.ValueString(), Type: m.Type.ValueString(), Value: m.Value.ValueString()}
	if !m.TTL.IsNull() && !m.TTL.IsUnknown() {
		v := m.TTL.ValueInt64()
		w.TTL = &v
	}
	if !m.Priority.IsNull() && !m.Priority.IsUnknown() {
		v := m.Priority.ValueInt64()
		w.Priority = &v
	}
	if !m.Proxied.IsNull() && !m.Proxied.IsUnknown() {
		v := m.Proxied.ValueBool()
		w.Proxied = &v
	}
	return w
}

// fromAPI copies the server's view into the model. Optional inputs the user left unset stay
// null, so a server default never shows up as a perpetual diff.
func fromAPI(m *dnsRecordModel, rec *zinnapi.DnsRecord) {
	m.ID = types.StringValue(rec.ID)
	m.Name = types.StringValue(rec.Name)
	m.Type = types.StringValue(rec.Type)
	m.Value = types.StringValue(rec.Value)
	m.TTL = types.Int64Value(rec.TTL)
	m.Owner = types.StringValue(rec.Owner)
	if !m.Priority.IsNull() || rec.Priority != nil {
		m.Priority = int64Ptr(rec.Priority)
	}
	if !m.Proxied.IsNull() {
		m.Proxied = boolPtr(rec.Proxied)
	}
}

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !needsKey(r.client, &resp.Diagnostics) {
		return
	}
	rec, err := r.client.CreateDNSRecord(ctx, plan.DomainID.ValueString(), toWrite(plan))
	if err != nil {
		resp.Diagnostics.AddError("Creating DNS record failed", err.Error())
		return
	}
	fromAPI(&plan, rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !needsKey(r.client, &resp.Diagnostics) {
		return
	}
	rec, err := r.client.DNSRecord(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if zinnapi.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading DNS record failed", err.Error())
		return
	}
	fromAPI(&state, rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !needsKey(r.client, &resp.Diagnostics) {
		return
	}
	rec, err := r.client.UpdateDNSRecord(ctx, state.DomainID.ValueString(), state.ID.ValueString(), toWrite(plan))
	if err != nil {
		resp.Diagnostics.AddError("Updating DNS record failed", err.Error())
		return
	}
	fromAPI(&plan, rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !needsKey(r.client, &resp.Diagnostics) {
		return
	}
	if err := r.client.DeleteDNSRecord(ctx, state.DomainID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting DNS record failed", err.Error())
	}
}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, recordID, ok := strings.Cut(req.ID, "/")
	if !ok || domainID == "" || recordID == "" {
		resp.Diagnostics.AddError("Invalid import id", "Use <domain_id>/<record_id>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), recordID)...)
}
