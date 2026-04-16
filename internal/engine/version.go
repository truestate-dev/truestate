package engine

import (
	"strconv"
	"strings"
	"unicode"
)

// versionGTE reports whether installed >= fixed using the Debian version
// comparison algorithm (Policy Manual §5.6.12).
//
// Debian version format: [epoch:]upstream[-revision]
//
// Comparison rules:
//  1. Epoch (integer before ':') — higher epoch wins. Missing epoch = 0.
//  2. Upstream version and revision — compared by alternating non-digit /
//     digit runs. Non-digit runs use modified ASCII order (~  < empty < A-Z
//     < a-z < other). Digit runs are compared numerically (leading zeros
//     ignored).
func versionGTE(installed, fixed string) bool {
	return compareVersions(installed, fixed) >= 0
}

// compareVersions returns negative / zero / positive like strings.Compare.
func compareVersions(a, b string) int {
	ae, aRest := splitEpoch(a)
	be, bRest := splitEpoch(b)

	if ae != be {
		if ae > be {
			return 1
		}
		return -1
	}

	aUpstream, aRev := splitRevision(aRest)
	bUpstream, bRev := splitRevision(bRest)

	if c := compareUpstream(aUpstream, bUpstream); c != 0 {
		return c
	}
	return compareUpstream(aRev, bRev)
}

// splitEpoch splits "epoch:rest" into (epoch, rest). Missing epoch → 0.
func splitEpoch(v string) (int, string) {
	if i := strings.IndexByte(v, ':'); i >= 0 {
		n, err := strconv.Atoi(v[:i])
		if err == nil {
			return n, v[i+1:]
		}
	}
	return 0, v
}

// splitRevision splits "upstream-revision" at the last '-'.
// If no '-', revision is "0".
func splitRevision(v string) (string, string) {
	i := strings.LastIndexByte(v, '-')
	if i < 0 {
		return v, "0"
	}
	return v[:i], v[i+1:]
}

// compareUpstream compares two upstream-version / revision strings using
// the dpkg alternating non-digit / digit run algorithm.
func compareUpstream(a, b string) int {
	for len(a) > 0 || len(b) > 0 {
		// Non-digit run
		an, aRem := nonDigitRun(a)
		bn, bRem := nonDigitRun(b)
		if c := compareNonDigit(an, bn); c != 0 {
			return c
		}
		a, b = aRem, bRem

		// Digit run
		ad, aRem := digitRun(a)
		bd, bRem := digitRun(b)
		if c := compareDigit(ad, bd); c != 0 {
			return c
		}
		a, b = aRem, bRem
	}
	return 0
}

func nonDigitRun(s string) (string, string) {
	for i, r := range s {
		if unicode.IsDigit(r) {
			return s[:i], s[i:]
		}
	}
	return s, ""
}

func digitRun(s string) (string, string) {
	for i, r := range s {
		if !unicode.IsDigit(r) {
			return s[:i], s[i:]
		}
	}
	return s, ""
}

// compareNonDigit compares two non-digit strings using dpkg character order:
// '~' < empty < letters < other characters (all by ASCII, with ~ as special).
func compareNonDigit(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	n := len(ra)
	if len(rb) < n {
		n = len(rb)
	}
	for i := 0; i < n; i++ {
		ca := dpkgOrder(ra[i])
		cb := dpkgOrder(rb[i])
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
	}
	// Remaining characters: empty string sorts before any character (except ~)
	if len(ra) < len(rb) {
		if dpkgOrder(rb[len(ra)]) < 0 { // next char in b is '~'
			return 1
		}
		return -1
	}
	if len(ra) > len(rb) {
		if dpkgOrder(ra[len(rb)]) < 0 { // next char in a is '~'
			return -1
		}
		return 1
	}
	return 0
}

// dpkgOrder returns the sort key for a character in a non-digit run.
// '~' → -1 (sorts before everything including empty string)
// letter → ASCII value (positive)
// other  → ASCII value + 256 (sorts after letters)
func dpkgOrder(r rune) int {
	if r == '~' {
		return -1
	}
	if unicode.IsLetter(r) {
		return int(r)
	}
	return int(r) + 256
}

// compareDigit compares two digit strings numerically.
func compareDigit(a, b string) int {
	// Strip leading zeros for numeric comparison
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	// Same length: lexicographic == numeric
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
