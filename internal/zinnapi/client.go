// Package zinnapi is the provider's small HTTP client for the Zinn Digital Platform API.
// Its request and response types are generated from openapi/openapi.yaml (models_gen.go).
package zinnapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API.
const DefaultBaseURL = "https://api.zinndigital.com"

// Client calls the API with an optional per-organisation API key.
type Client struct {
	BaseURL   string
	APIKey    string
	Org       string
	UserAgent string
	HTTP      *http.Client
}

// New returns a client with sane timeouts.
func New(baseURL, apiKey, org, userAgent string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		Org:       org,
		UserAgent: userAgent,
		HTTP:      &http.Client{Timeout: 60 * time.Second},
	}
}

// Error is a non-2xx answer, carrying the API's error envelope verbatim.
type Error struct {
	Status int
	Body   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("zinn api: HTTP %d: %s", e.Status, e.Body)
}

// NotFound reports whether err is an HTTP 404 from the API.
func NotFound(err error) bool {
	apiErr, ok := err.(*Error)
	return ok && apiErr.Status == http.StatusNotFound
}

// Do sends one request; body (if any) is JSON-encoded and out (if any) decoded from JSON.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if c.Org != "" {
		req.Header.Set("X-Zinn-Org", c.Org)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &Error{Status: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// Regions lists the public hosting regions (no credential needed).
func (c *Client) Regions(ctx context.Context) ([]PublicHostingRegion, error) {
	var out struct {
		Data []PublicHostingRegion `json:"data"`
	}
	err := c.Do(ctx, http.MethodGet, "/v1/catalog/regions", nil, nil, &out)
	return out.Data, err
}

// Plans lists the public plan catalogue (no credential needed).
func (c *Client) Plans(ctx context.Context, productLine string) ([]CatalogPlan, error) {
	q := url.Values{}
	if productLine != "" {
		q.Set("product_line", productLine)
	}
	var out CatalogPlanList
	err := c.Do(ctx, http.MethodGet, "/v1/catalog/plans", q, nil, &out)
	return out.Data, err
}

// pages walks a cursor-paginated collection to its end.
func pages[T any](ctx context.Context, c *Client, path string, q url.Values, read func([]byte) ([]T, PageMeta, error)) ([]T, error) {
	var all []T
	if q == nil {
		q = url.Values{}
	}
	q.Set("limit", "200")
	for {
		var raw json.RawMessage
		if err := c.Do(ctx, http.MethodGet, path, q, nil, &raw); err != nil {
			return nil, err
		}
		items, page, err := read(raw)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !page.HasMore || page.NextCursor == nil || *page.NextCursor == "" {
			return all, nil
		}
		q.Set("cursor", *page.NextCursor)
	}
}

// Sites lists every site the credential's organisation can see.
func (c *Client) Sites(ctx context.Context) ([]Site, error) {
	return pages(ctx, c, "/v1/sites", nil, func(b []byte) ([]Site, PageMeta, error) {
		var p SitePage
		err := json.Unmarshal(b, &p)
		return p.Data, p.Page, err
	})
}

// Domains lists every domain the credential's organisation can see.
func (c *Client) Domains(ctx context.Context) ([]Domain, error) {
	return pages(ctx, c, "/v1/domains", nil, func(b []byte) ([]Domain, PageMeta, error) {
		var p DomainPage
		err := json.Unmarshal(b, &p)
		return p.Data, p.Page, err
	})
}

// DNSRecords lists a domain's records.
func (c *Client) DNSRecords(ctx context.Context, domainID string) ([]DnsRecord, error) {
	path := "/v1/domains/" + url.PathEscape(domainID) + "/dns/records"
	return pages(ctx, c, path, nil, func(b []byte) ([]DnsRecord, PageMeta, error) {
		var p DnsRecordList
		err := json.Unmarshal(b, &p)
		return p.Data, p.Page, err
	})
}

// DNSRecord finds one record by id (the API has no single-record GET), or a 404 Error.
func (c *Client) DNSRecord(ctx context.Context, domainID, recordID string) (*DnsRecord, error) {
	records, err := c.DNSRecords(ctx, domainID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == recordID {
			return &records[i], nil
		}
	}
	return nil, &Error{Status: http.StatusNotFound, Body: "dns record " + recordID + " not found"}
}

// CreateDNSRecord creates a record.
func (c *Client) CreateDNSRecord(ctx context.Context, domainID string, w DnsRecordWrite) (*DnsRecord, error) {
	var out DnsRecord
	err := c.Do(ctx, http.MethodPost, "/v1/domains/"+url.PathEscape(domainID)+"/dns/records", nil, w, &out)
	return &out, err
}

// UpdateDNSRecord replaces a record.
func (c *Client) UpdateDNSRecord(ctx context.Context, domainID, recordID string, w DnsRecordWrite) (*DnsRecord, error) {
	var out DnsRecord
	path := "/v1/domains/" + url.PathEscape(domainID) + "/dns/records/" + url.PathEscape(recordID)
	err := c.Do(ctx, http.MethodPut, path, nil, w, &out)
	return &out, err
}

// DeleteDNSRecord deletes a record; a record already gone is not an error.
func (c *Client) DeleteDNSRecord(ctx context.Context, domainID, recordID string) error {
	path := "/v1/domains/" + url.PathEscape(domainID) + "/dns/records/" + url.PathEscape(recordID)
	err := c.Do(ctx, http.MethodDelete, path, nil, nil, nil)
	if NotFound(err) {
		return nil
	}
	return err
}
