// Package ubuntu ingests package vulnerability data from the Ubuntu Security Tracker.
//
// Data source: https://ubuntu.com/security/cves.json (paginated)
// USN feed:    https://ubuntu.com/security/notices/feed.atom
//
// Ubuntu CVE JSON structure (per CVE):
//   {
//     "id": "CVE-...",
//     "description": "...",
//     "ubuntu_description": "...",
//     "published_at": "...",
//     "packages": [
//       {
//         "name": "...",
//         "source": "...",
//         "statuses": [
//           {
//             "release_codename": "jammy",
//             "status": "released|needed|not-affected|deferred|ignored|pending",
//             "description": "...",
//             "pocket": "security"
//           }
//         ]
//       }
//     ]
//   }
package ubuntu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

const baseURL = "https://ubuntu.com/security/cves.json"

type packageStatus struct {
	ReleaseCodename string `json:"release_codename"`
	Status          string `json:"status"`
	Description     string `json:"description"`
	Pocket          string `json:"pocket"`
}

type cvePackage struct {
	Name     string          `json:"name"`
	Statuses []packageStatus `json:"statuses"`
}

type cveEntry struct {
	ID                 string       `json:"id"`
	Description        string       `json:"description"`
	UbuntuDescription  string       `json:"ubuntu_description"`
	PublishedAt        string       `json:"published_at"`
	Packages           []cvePackage `json:"packages"`
}

type page struct {
	CVEs   []cveEntry `json:"cves"`
	Offset int        `json:"offset"`
	Limit  int        `json:"limit"`
	Total  int        `json:"total"`
}

// Result holds parsed assertions from one sync run.
type Result struct {
	Vulnerabilities []model.Vulnerability
	Assertions      []model.Assertion
}

// Fetch downloads and parses all pages of the Ubuntu CVE feed.
func Fetch(ctx context.Context) (*Result, error) {
	result := &Result{}
	seen := map[string]bool{}
	offset := 0
	limit := 500

	for {
		url := fmt.Sprintf("%s?limit=%d&offset=%d", baseURL, limit, offset)
		slog.Info("ubuntu: fetching page", "offset", offset, "limit", limit)

		p, err := fetchPage(ctx, url)
		if err != nil {
			return nil, err
		}

		parsePage(p, result, seen)

		offset += limit
		if offset >= p.Total {
			break
		}
	}

	slog.Info("ubuntu: fetch complete",
		"vulnerabilities", len(result.Vulnerabilities),
		"assertions", len(result.Assertions),
	)
	return result, nil
}

func fetchPage(ctx context.Context, url string) (*page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ubuntu: build request: %w", err)
	}
	req.Header.Set("User-Agent", "truestate/0.1 (security research tool)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ubuntu: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ubuntu: unexpected status %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ubuntu: read body: %w", err)
	}

	var p page
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("ubuntu: parse page: %w", err)
	}
	return &p, nil
}

// ParsePage parses a single page of CVE data. Exported for testing.
func ParsePage(data []byte) (*Result, error) {
	var p page
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("ubuntu: parse page: %w", err)
	}
	result := &Result{}
	parsePage(&p, result, map[string]bool{})
	return result, nil
}

func parsePage(p *page, result *Result, seen map[string]bool) {
	now := time.Now().UTC()
	for _, cve := range p.CVEs {
		if !seen[cve.ID] {
			desc := cve.UbuntuDescription
			if desc == "" {
				desc = cve.Description
			}
			result.Vulnerabilities = append(result.Vulnerabilities, model.Vulnerability{
				CVEID:   cve.ID,
				Summary: desc,
			})
			seen[cve.ID] = true
		}

		for _, pkg := range cve.Packages {
			for _, st := range pkg.Statuses {
				result.Assertions = append(result.Assertions, model.Assertion{
					Source:      model.SourceUbuntu,
					CVEID:       cve.ID,
					PackageName: pkg.Name,
					Platform:    model.PlatformUbuntu,
					Release:     st.ReleaseCodename,
					Status:      mapStatus(st.Status),
					FetchedAt:   now,
				})
			}
		}
	}
}

// mapStatus converts Ubuntu tracker status strings to our AssertionStatus type.
func mapStatus(s string) model.AssertionStatus {
	switch s {
	case "released":
		return model.AssertionFixed
	case "needed", "pending":
		return model.AssertionAffected
	case "not-affected":
		return model.AssertionNotAffected
	case "deferred", "ignored":
		return model.AssertionUnderReview
	default:
		return model.AssertionUnderReview
	}
}
