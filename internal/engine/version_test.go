package engine

import "testing"

// Test cases sourced from real Debian/Ubuntu version pairs and
// dpkg policy §5.6.12 examples.
var versionTests = []struct {
	a, b string
	want int // -1 a<b, 0 a==b, 1 a>b
}{
	// Equal
	{"1.0", "1.0", 0},
	{"1:1.0", "1:1.0", 0},
	{"2:3.4.5-1", "2:3.4.5-1", 0},

	// Epoch dominates
	{"1:0.9", "0:2.0", 1},
	{"0:2.0", "1:0.9", -1},
	{"2:1.0", "1:99.0", 1},

	// Missing epoch treated as 0
	{"1.0", "1:0.1", -1},
	{"1:0.1", "1.0", 1},

	// Upstream version: numeric runs
	{"1.10", "1.9", 1},
	{"1.9", "1.10", -1},
	{"1.0.0", "1.0", 1},  // longer is greater when prefix matches

	// Revision
	{"1.0-2", "1.0-10", -1},
	{"1.0-10", "1.0-2", 1},
	{"1.0-1", "1.0-1", 0},

	// Tilde sorts before everything (pre-release)
	{"1.0~rc1", "1.0", -1},
	{"1.0", "1.0~rc1", 1},
	{"1.0~beta1", "1.0~rc1", -1}, // beta < rc alphabetically

	// Real Ubuntu openssl versions (the motivating case)
	{"3.0.2-0ubuntu1.10", "3.0.2-0ubuntu1.12", -1}, // installed < fixed → vulnerable
	{"3.0.2-0ubuntu1.12", "3.0.2-0ubuntu1.12", 0},  // exactly fixed → not vulnerable
	{"3.0.2-0ubuntu1.15", "3.0.2-0ubuntu1.12", 1},  // newer than fix → not vulnerable

	// Real Debian curl versions
	{"7.81.0-1ubuntu1.13", "7.81.0-1ubuntu1.15", -1},
	{"7.81.0-1ubuntu1.15", "7.81.0-1ubuntu1.15", 0},

	// Leading zeros in digit runs
	{"1.01", "1.1", 0},
	{"1.001", "1.1", 0},

	// Alpha characters in non-digit runs
	{"1.0a", "1.0b", -1},
	{"1.0b", "1.0a", 1},
	{"1.0+deb11u1", "1.0+deb11u2", -1},
}

func TestCompareVersions(t *testing.T) {
	for _, tc := range versionTests {
		got := sign(compareVersions(tc.a, tc.b))
		if got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestVersionGTE(t *testing.T) {
	cases := []struct {
		installed, fixed string
		gte              bool
	}{
		{"3.0.2-0ubuntu1.12", "3.0.2-0ubuntu1.12", true},  // at fix
		{"3.0.2-0ubuntu1.15", "3.0.2-0ubuntu1.12", true},  // above fix
		{"3.0.2-0ubuntu1.10", "3.0.2-0ubuntu1.12", false}, // below fix → vulnerable
		{"1.0~rc1", "1.0", false},                          // pre-release < release
	}
	for _, tc := range cases {
		got := versionGTE(tc.installed, tc.fixed)
		if got != tc.gte {
			t.Errorf("versionGTE(%q, %q) = %v, want %v", tc.installed, tc.fixed, got, tc.gte)
		}
	}
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}
