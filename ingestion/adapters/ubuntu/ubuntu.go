// Package ubuntu ingests package vulnerability data from Ubuntu OVAL XML files.
//
// Data source: https://security-metadata.canonical.com/oval/com.ubuntu.<release>.usn.oval.xml.bz2
//
// OVAL (Open Vulnerability and Assessment Language) files are published per
// Ubuntu release codename (focal, jammy, noble, ...). Each file contains all
// USN-referenced CVEs and their affected package versions for that release.
//
// Structure relevant to parsing:
//   - CVE IDs are in <metadata><reference source="CVE" ref_id="CVE-YYYY-NNNNN">
//   - Fixed package versions are in <metadata><description> as lines:
//       packagename - fixedversion
//     appearing after a "following package versions:" marker.
package ubuntu

import (
	"compress/bzip2"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

// SupportedReleases lists the Ubuntu releases we sync by default.
var SupportedReleases = []string{"focal", "jammy", "noble"}

const ovalURLTemplate = "https://security-metadata.canonical.com/oval/com.ubuntu.%s.usn.oval.xml.bz2"

// Result holds parsed assertions from one sync run.
type Result struct {
	Vulnerabilities []model.Vulnerability
	Assertions      []model.Assertion
}

// Fetch downloads and parses OVAL data for all supported Ubuntu releases.
func Fetch(ctx context.Context) (*Result, error) {
	result := &Result{}
	for _, release := range SupportedReleases {
		slog.Info("ubuntu: fetching OVAL", "release", release)
		r, err := fetchRelease(ctx, release)
		if err != nil {
			return nil, fmt.Errorf("ubuntu: %s: %w", release, err)
		}
		result.Vulnerabilities = append(result.Vulnerabilities, r.Vulnerabilities...)
		result.Assertions = append(result.Assertions, r.Assertions...)
	}
	slog.Info("ubuntu: fetch complete",
		"releases", len(SupportedReleases),
		"vulnerabilities", len(result.Vulnerabilities),
		"assertions", len(result.Assertions),
	)
	return result, nil
}

// fetchRelease fetches and parses the OVAL file for a single Ubuntu release.
func fetchRelease(ctx context.Context, release string) (*Result, error) {
	url := fmt.Sprintf(ovalURLTemplate, release)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "truestate/0.1 (security research tool)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	return parseOVAL(bzip2.NewReader(resp.Body), release)
}

// OVAL XML types — we only parse the fields we need.

type ovalRoot struct {
	Definitions []ovalDefinition `xml:"definitions>definition"`
}

type ovalDefinition struct {
	ID       string       `xml:"id,attr"`
	Metadata ovalMetadata `xml:"metadata"`
}

type ovalMetadata struct {
	Title       string          `xml:"title"`
	Description string          `xml:"description"`
	References  []ovalReference `xml:"reference"`
	Advisory    ovalAdvisory    `xml:"advisory"`
}

// ovalReference captures <reference source="CVE" ref_id="CVE-YYYY-NNNNN"> elements.
type ovalReference struct {
	Source string `xml:"source,attr"`
	RefID  string `xml:"ref_id,attr"`
}

// ovalAdvisory is kept for fallback CVE extraction.
type ovalAdvisory struct {
	CVEs []ovalCVE `xml:"cve"`
}

type ovalCVE struct {
	Value string `xml:",chardata"`
}

// pkgEntry matches "packagename - version" pairs anywhere in a string.
// Descriptions contain all packages on one line separated by spaces, e.g.:
//   "curl - 7.81.0-1ubuntu1.2 libcurl4 - 7.81.0-1ubuntu1.2 Available with..."
// Package names follow dpkg naming (a-z0-9.+-).
// Versions always start with a digit (or epoch:digit), so we anchor on \d.
var pkgEntry = regexp.MustCompile(`([a-z0-9][a-z0-9.+\-]*)\s+-\s+(\d[^\s]*)`)

// parseOVAL parses the bzip2-decompressed OVAL XML stream for one release.
func parseOVAL(r io.Reader, release string) (*Result, error) {
	var root ovalRoot
	if err := xml.NewDecoder(r).Decode(&root); err != nil {
		return nil, fmt.Errorf("parse OVAL XML: %w", err)
	}

	result := &Result{}
	seen := map[string]bool{}
	now := time.Now().UTC()

	for _, def := range root.Definitions {
		// Collect CVE IDs from <reference source="CVE"> elements (primary).
		var cveIDs []string
		for _, ref := range def.Metadata.References {
			if strings.EqualFold(ref.Source, "CVE") {
				id := strings.TrimSpace(ref.RefID)
				if strings.HasPrefix(id, "CVE-") {
					cveIDs = append(cveIDs, id)
				}
			}
		}
		// Fallback: also collect from <advisory><cve> text content.
		if len(cveIDs) == 0 {
			for _, c := range def.Metadata.Advisory.CVEs {
				id := strings.TrimSpace(c.Value)
				if id != "" && strings.HasPrefix(id, "CVE-") {
					cveIDs = append(cveIDs, id)
				}
			}
		}
		if len(cveIDs) == 0 {
			continue
		}

		// Record vulnerability metadata for each CVE (de-duplicated across releases).
		for _, cveID := range cveIDs {
			if !seen[cveID] {
				result.Vulnerabilities = append(result.Vulnerabilities, model.Vulnerability{
					CVEID:   cveID,
					Summary: strings.TrimSpace(def.Metadata.Description),
				})
				seen[cveID] = true
			}
		}

		// Extract affected packages and their fixed versions from the description text.
		// Example description excerpt:
		//   "...the following package versions:
		//     libcurl4 - 7.81.0-1ubuntu1.1
		//     curl - 7.81.0-1ubuntu1.1"
		pkgs := parseDescriptionPackages(def.Metadata.Description)
		for _, pkg := range pkgs {
			for _, cveID := range cveIDs {
				result.Assertions = append(result.Assertions, model.Assertion{
					Source:       model.SourceUbuntu,
					CVEID:        cveID,
					PackageName:  pkg.name,
					Platform:     model.PlatformUbuntu,
					Release:      release,
					Status:       model.AssertionFixed,
					FixedVersion: pkg.fixedVersion,
					FetchedAt:    now,
				})
			}
		}
	}

	slog.Info("ubuntu: parsed OVAL", "release", release,
		"definitions", len(root.Definitions),
		"assertions", len(result.Assertions),
	)
	return result, nil
}

type pkgAssertion struct {
	name         string
	fixedVersion string
}

// parseDescriptionPackages extracts package name + fixed version pairs from
// an OVAL definition description. Ubuntu OVAL descriptions list all packages
// on a single space-separated line after a "following package versions:" marker,
// in the form:
//
//	"... following package versions:  pkg1 - ver1 pkg2 - ver2 Available with Ubuntu Pro"
//
// We locate the marker, then extract all (name, version) pairs from the text that
// follows using a regex that requires versions to start with a digit.
func parseDescriptionPackages(description string) []pkgAssertion {
	// Narrow to the section after the package list header.
	const marker = "following package versions:"
	idx := strings.Index(strings.ToLower(description), marker)
	if idx < 0 {
		return nil
	}
	section := description[idx+len(marker):]

	// Stop at any "Available with Ubuntu Pro" suffix.
	if stop := strings.Index(section, "Available with Ubuntu Pro"); stop >= 0 {
		section = section[:stop]
	}

	matches := pkgEntry.FindAllStringSubmatch(section, -1)
	var out []pkgAssertion
	for _, m := range matches {
		out = append(out, pkgAssertion{name: m[1], fixedVersion: m[2]})
	}
	return out
}
