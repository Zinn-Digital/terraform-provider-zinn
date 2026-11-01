package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var factories = map[string]func() (tfprotov6.ProviderServer, error){
	"zinn": providerserver.NewProtocol6WithError(New("test")()),
}

// fakeAPI is an in-memory stand-in for the endpoints the provider calls, shaped by the spec.
type fakeAPI struct {
	mu      sync.Mutex
	records map[string]map[string]any
	seq     int
	auth    []string
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auth = append(f.auth, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
	w.Header().Set("Content-Type", "application/json")
	const recs = "/v1/domains/dom-1/dns/records"
	switch {
	case r.URL.Path == "/v1/catalog/regions":
		io.WriteString(w, `{"data":[{"slug":"helsinki","city":"Helsinki","country":"FI"}]}`)
	case r.URL.Path == "/v1/catalog/plans":
		if r.URL.Query().Get("product_line") != "agency" {
			http.Error(w, `{"error":"want product_line=agency"}`, 400)
			return
		}
		io.WriteString(w, `{"data":[{"id":"p1","code":"agency_start","name":"Start","product_line":"agency","plan_version_id":"v1","plan_version":1,"recommended":true,"sort_order":1,"prices":[],"entitlements":{}}]}`)
	case r.URL.Path == "/v1/domains":
		if r.URL.Query().Get("cursor") == "" {
			io.WriteString(w, `{"data":[{"id":"dom-1","fqdn":"a.example","status":"active","ownership_mode":"external","auto_renew":false,"nameservers":["ns1.example"]}],"page":{"has_more":true,"next_cursor":"c2"}}`)
		} else {
			io.WriteString(w, `{"data":[{"id":"dom-2","fqdn":"b.example","status":"active","ownership_mode":"external","auto_renew":true,"nameservers":[]}],"page":{"has_more":false}}`)
		}
	case r.URL.Path == recs && r.Method == http.MethodGet:
		out := []any{}
		for _, v := range f.records {
			out = append(out, v)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": out, "page": map[string]any{"has_more": false}, "supported_types": []string{"A"}})
	case r.URL.Path == recs && r.Method == http.MethodPost, strings.HasPrefix(r.URL.Path, recs+"/") && r.Method == http.MethodPut:
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		id := strings.TrimPrefix(r.URL.Path, recs+"/")
		if r.Method == http.MethodPost {
			f.seq++
			id = fmt.Sprintf("rec-%d", f.seq)
		} else if _, ok := f.records[id]; !ok {
			http.Error(w, `{"error":"not found"}`, 404)
			return
		}
		if body["ttl"] == nil {
			body["ttl"] = 300
		}
		body["id"], body["owner"] = id, "customer"
		f.records[id] = body
		if r.Method == http.MethodPost {
			w.WriteHeader(201)
		}
		json.NewEncoder(w).Encode(body)
	case strings.HasPrefix(r.URL.Path, recs+"/") && r.Method == http.MethodDelete:
		delete(f.records, strings.TrimPrefix(r.URL.Path, recs+"/"))
		w.WriteHeader(204)
	default:
		http.Error(w, `{"error":"unexpected `+r.Method+" "+r.URL.Path+`"}`, 500)
	}
}

func providerBlock(url, key string) string {
	return fmt.Sprintf("provider \"zinn\" {\n  api_url = %q\n  api_key = %q\n}\n", url, key)
}

func TestPublicDataSourcesNeedNoKey(t *testing.T) {
	api := &fakeAPI{records: map[string]map[string]any{}}
	srv := httptest.NewServer(api)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config: providerBlock(srv.URL, "") + `
data "zinn_regions" "all" {}
data "zinn_plans" "agency" { product_line = "agency" }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.zinn_regions.all", "regions.0.slug", "helsinki"),
				resource.TestCheckResourceAttr("data.zinn_plans.agency", "plans.#", "1"),
				resource.TestCheckResourceAttr("data.zinn_plans.agency", "plans.0.code", "agency_start"),
				resource.TestCheckResourceAttr("data.zinn_plans.agency", "plans.0.plan_version_id", "v1"),
			),
		}},
	})
	for _, line := range api.auth {
		if strings.Contains(line, "Bearer") {
			t.Fatalf("a public call carried a credential: %s", line)
		}
	}
}

func TestDomainsFollowsEveryPage(t *testing.T) {
	srv := httptest.NewServer(&fakeAPI{records: map[string]map[string]any{}})
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config: providerBlock(srv.URL, "zk_test") + `data "zinn_domains" "all" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.zinn_domains.all", "domains.#", "2"),
				resource.TestCheckResourceAttr("data.zinn_domains.all", "domains.1.fqdn", "b.example"),
			),
		}},
	})
}

func TestKeyedDataSourceWithoutKeyFailsClearly(t *testing.T) {
	srv := httptest.NewServer(&fakeAPI{records: map[string]map[string]any{}})
	defer srv.Close()
	t.Setenv("ZINN_API_KEY", "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config:      providerBlock(srv.URL, "") + `data "zinn_sites" "all" {}`,
			ExpectError: regexpMust("API key required"),
		}},
	})
}

func TestDNSRecordLifecycle(t *testing.T) {
	api := &fakeAPI{records: map[string]map[string]any{}}
	srv := httptest.NewServer(api)
	defer srv.Close()
	cfg := func(value string) string {
		return providerBlock(srv.URL, "zk_test") + fmt.Sprintf(`
resource "zinn_dns_record" "www" {
  domain_id = "dom-1"
  name      = "www"
  type      = "A"
  value     = %q
}`, value)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg("192.0.2.1"), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("zinn_dns_record.www", "id", "rec-1"),
				resource.TestCheckResourceAttr("zinn_dns_record.www", "ttl", "300"),
				resource.TestCheckResourceAttr("zinn_dns_record.www", "owner", "customer"),
			)},
			{Config: cfg("192.0.2.2"), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("zinn_dns_record.www", "id", "rec-1"),
				resource.TestCheckResourceAttr("zinn_dns_record.www", "value", "192.0.2.2"),
			)},
			{ResourceName: "zinn_dns_record.www", ImportState: true, ImportStateVerify: true, ImportStateId: "dom-1/rec-1"},
		},
		CheckDestroy: func(_ *terraformState) error {
			if len(api.records) != 0 {
				return fmt.Errorf("records left after destroy: %v", api.records)
			}
			return nil
		},
	})
	for _, line := range api.auth {
		if !strings.HasSuffix(line, "Bearer zk_test") {
			t.Fatalf("a DNS call went without the key: %s", line)
		}
	}
}
