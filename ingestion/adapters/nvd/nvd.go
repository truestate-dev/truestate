// Package nvd fetches CVSS scores from the NVD (National Vulnerability Database)
// REST API 2.0 and returns enrichment records for existing vulnerabilities.
//
// API reference: https://nvd.nist.gov/developers/vulnerabilities
//
// Rate limits:
//   - Without API key:  5 requests / 30-second rolling window  (~10/min)
//   - With NVD_API_KEY: 50 requests / 30-second rolling window (~100/min)
//
// The adapter pages through the full NVD CVE dataset, picking up the best
// available CVSS score (v3.1 > v3.0 > v2) for each CVE. Only CVEs that
// already exist in our vulnerabilities table are updated (the caller filters
// via BulkUpdateVulnerabilityCVSS).
package nvd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gitea.local.vjinx.de/truestate/truestate/internal/db"
)

const (
	apiBase      = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	pageSize     = 500
	sleepNoKey   = 6200 * time.Millisecond // ~5 req/30s
	sleepWithKey = 700 * time.Millisecond  // ~50 req/30s
)

// ─── NVD API response structs ────────────────────────────────────────────────

type nvdResponse struct {
	TotalResults    int      `json:"totalResults"`
	StartIndex      int      `json:"startIndex"`
	Vulnerabilities []nvdCVE `json:"vulnerabilities"`
}

type nvdCVE struct {
	CVE nvdCVEDetail `json:"cve"`
}

type nvdCVEDetail struct {
	ID      string     `json:"id"`
	Metrics nvdMetrics `json:"metrics"`
}

type nvdMetrics struct {
	V31 []nvdMetricEntry `json:"cvssMetricV31"`
	V30 []nvdMetricEntry `json:"cvssMetricV30"`
	V2  []nvdMetricV2    `json:"cvssMetricV2"`
}

type nvdCVSSData struct {
	BaseScore    float64 `json:"baseScore"`
	BaseSeverity string  `json:"baseSeverity"`
	VectorString string  `json:"vectorString"`
}

// nvdMetricEntry covers cvssMetricV31 and cvssMetricV30 (severity inside cvssData).
type nvdMetricEntry struct {
	Type     string      `json:"type"`
	CVSSData nvdCVSSData `json:"cvssData"`
}

// nvdMetricV2 has severity at the top level, not inside cvssData.
type nvdMetricV2 struct {
	Type         string      `json:"type"`
	BaseSeverity string      `json:"baseSeverity"`
	CVSSData     nvdCVSSData `json:"cvssData"`
}

// ─── Public API ──────────────────────────────────────────────────────────────

// Fetch pages through the NVD CVE API and returns all CVSS enrichment records.
// Progress is logged every 10 pages.
func Fetch(ctx context.Context) ([]db.CVSSRecord, error) {
	apiKey := os.Getenv("NVD_API_KEY")
	delay := sleepNoKey
	if apiKey != "" {
		delay = sleepWithKey
		slog.Info("nvd: using API key")
	} else {
		slog.Info("nvd: no API key — rate limited to 5 req/30s; set NVD_API_KEY to speed up")
	}

	client := &http.Client{Timeout: 120 * time.Second}
	var records []db.CVSSRecord
	startIndex := 0
	total := -1
	page := 0

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		url := fmt.Sprintf("%s?startIndex=%d&resultsPerPage=%d", apiBase, startIndex, pageSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("nvd: build request: %w", err)
		}
		req.Header.Set("User-Agent", "truestate/0.1 (security research tool)")
		if apiKey != "" {
			req.Header.Set("apiKey", apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("nvd: fetch page %d: %w", page, err)
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == 429 ||
			resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			slog.Warn("nvd: backing off 35s", "page", page, "status", resp.StatusCode)
			select {
			case <-time.After(35 * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue // retry same page
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("nvd: unexpected status %d: %s", resp.StatusCode, body)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("nvd: read page %d: %w", page, err)
		}

		var nvdResp nvdResponse
		if err := json.Unmarshal(body, &nvdResp); err != nil {
			return nil, fmt.Errorf("nvd: parse page %d: %w", page, err)
		}

		if total < 0 {
			total = nvdResp.TotalResults
			slog.Info("nvd: starting sync", "total_cves", total)
		}

		for _, item := range nvdResp.Vulnerabilities {
			r := extractCVSS(item.CVE)
			if r.Score > 0 {
				records = append(records, r)
			}
		}

		startIndex += len(nvdResp.Vulnerabilities)
		page++

		if page%10 == 0 {
			slog.Info("nvd: progress", "page", page, "fetched", startIndex, "total", total, "records_with_cvss", len(records))
		}

		if startIndex >= total || len(nvdResp.Vulnerabilities) == 0 {
			break
		}

		// Respect rate limit between pages.
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	slog.Info("nvd: fetch complete", "pages", page, "total_cves", startIndex, "records_with_cvss", len(records))
	return records, nil
}

// extractCVSS picks the best available CVSS score for a CVE (v3.1 > v3.0 > v2).
func extractCVSS(cve nvdCVEDetail) db.CVSSRecord {
	r := db.CVSSRecord{CVEID: cve.ID}

	if best := pickV3(cve.Metrics.V31); best != nil {
		r.Score = best.CVSSData.BaseScore
		r.Severity = best.CVSSData.BaseSeverity
		r.Vector = best.CVSSData.VectorString
		return r
	}
	if best := pickV3(cve.Metrics.V30); best != nil {
		r.Score = best.CVSSData.BaseScore
		r.Severity = best.CVSSData.BaseSeverity
		r.Vector = best.CVSSData.VectorString
		return r
	}
	if best := pickV2(cve.Metrics.V2); best != nil {
		r.Score = best.CVSSData.BaseScore
		r.Severity = best.BaseSeverity
		r.Vector = best.CVSSData.VectorString
		return r
	}
	return r
}

func pickV3(entries []nvdMetricEntry) *nvdMetricEntry {
	for i := range entries {
		if entries[i].Type == "Primary" {
			return &entries[i]
		}
	}
	if len(entries) > 0 {
		return &entries[0]
	}
	return nil
}

func pickV2(entries []nvdMetricV2) *nvdMetricV2 {
	for i := range entries {
		if entries[i].Type == "Primary" {
			return &entries[i]
		}
	}
	if len(entries) > 0 {
		return &entries[0]
	}
	return nil
}
