package version

import "testing"

func TestNewer(t *testing.T) {
	for _, tc := range []struct {
		latest, current string
		want            bool
	}{
		{"v1.10.0", "1.9.9", true},
		{"1.2.3", "v1.2.3", false},
		{"1.2.3", "1.2.3-rc.1", true},
		{"1.2.3", "1.2.3+build", false},
		{"1.2.3", "2.0.0", false},
		{"1.2.3-rc.1", "1.0.0", false},
		{"1.2.3", "dev", false},
		{"1.2.3", "v1.2.3-2-gabcd123", false},
		{"1.2.3", "v1.2.3-dirty", false},
		{"1.02.3", "1.0.0", false},
	} {
		if got := Newer(tc.latest, tc.current); got != tc.want {
			t.Errorf("Newer(%q, %q) = %v", tc.latest, tc.current, got)
		}
	}
}

func TestAndroidVersionCode(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int64
	}{
		{"v1.2.3", 1002003}, {"1.10.0", 1010000}, {"v2.0.0", 2000000},
		{"0.0.1", 1}, {"v2099.999.999", 2099999999},
	} {
		got, err := AndroidVersionCode(tc.value)
		if err != nil || got != tc.want {
			t.Errorf("AndroidVersionCode(%q) = %d, %v", tc.value, got, err)
		}
	}
	for _, value := range []string{
		"dev", "v1.2.3-rc.1", "v1.2.3+build", "v1.1000.0", "v2100.0.0", "v01.2.3", "v0.0.0",
		"v18446744073709551615.0.0", "v18446744073709551616.0.0",
	} {
		if _, err := AndroidVersionCode(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}
