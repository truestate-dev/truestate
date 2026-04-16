// Package debian ingests package vulnerability data from the Debian Security Tracker.
//
// Data source: https://security-tracker.debian.org/tracker/data/json
//
// Feed structure:
//   {
//     "<package>": {
//       "<cve-id>": {
//         "description": "...",
//         "releases": {
//           "<release>": {
//             "status": "resolved|open|undetermined",
//             "fixed_version": "...",
//             "urgency": "..."
//           }
//         }
//       }
//     }
//   }
package debian

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

const feedURL = "https://security-tracker.debian.org/tracker/data/json"

// releaseEntry is one release block inside a CVE entry.
type releaseEntry struct {
	Status       string `json:"status"`
	Repositories map[string]string `json:"repositories"`
	FixedVersion string `json:"fixed_version"`
	Urgency      string `json:"urgency"`
}

// cveEntry is one CVE block for a package.
type cveEntry struct {
	Description string                  `json:"description"`
	Releases    map[string]releaseEntry `json:"releases"`
}

// feed is the top-level structure: package → cve → cveEntry
type feed map[string]map[string]cveEntry

// Result holds the parsed assertions from one sync run.
type Result struct {
	Vulnerabilities []model.Vulnerability
	Assertions      []model.Assertion
}

// Fetch downloads and parses the Debian security tracker JSON feed.
func Fetch(ctx context.Context) (*Result, error) {
	slog.Info("debian: fetching feed", "url", feedURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("debian: build request: %w", err)
	}
	req.Header.Set("User-Agent", "truestate/0.1 (security research tool)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("debian: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("debian: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("debian: read body: %w", err)
	}

	return Parse(body)
}

// Parse parses the raw Debian JSON feed bytes into assertions.
// Exported so it can be tested independently of the HTTP layer.
func Parse(data []byte) (*Result, error) {
	var f feed
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("debian: parse feed: %w", err)
	}

	result := &Result{}
	seen := map[string]bool{}
	now := time.Now().UTC()

	for pkgName, cves := range f {
		for cveID, entry := range cves {
			if !seen[cveID] {
				result.Vulnerabilities = append(result.Vulnerabilities, model.Vulnerability{
					CVEID:   cveID,
					Summary: entry.Description,
				})
				seen[cveID] = true
			}

			for release, rel := range entry.Releases {
				status := mapStatus(rel.Status)
				result.Assertions = append(result.Assertions, model.Assertion{
					Source:       model.SourceDebian,
					CVEID:        cveID,
					PackageName:  pkgName,
					Platform:     model.PlatformDebian,
					Release:      release,
					Status:       status,
					FixedVersion: rel.FixedVersion,
					FetchedAt:    now,
				})
			}
		}
	}

	slog.Info("debian: parsed feed",
		"packages", len(f),
		"vulnerabilities", len(result.Vulnerabilities),
		"assertions", len(result.Assertions),
	)
	return result, nil
}

// mapStatus converts Debian tracker status strings to our AssertionStatus type.
func mapStatus(s string) model.AssertionStatus {
	switch s {
	case "resolved":
		return model.AssertionFixed
	case "open":
		return model.AssertionAffected
	case "undetermined":
		return model.AssertionUnderReview
	default:
		return model.AssertionUnderReview
	}
}
