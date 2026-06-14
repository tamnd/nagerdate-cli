package nagerdate

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring (mint, resolve), which need no network. The client's
// HTTP behaviour is covered in nagerdate_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "nagerdate" {
		t.Errorf("Scheme = %q, want nagerdate", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "nagerdate" {
		t.Errorf("Identity.Binary = %q, want nagerdate", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in  string
		typ string
		id  string
	}{
		{"US", "country", "US"},
		{"GB", "country", "GB"},
		{"FR", "country", "FR"},
		{"2024", "date", "2024"},
		{"2024-01-01", "date", "2024-01-01"},
		{"new year", "query", "new year"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("country", "US")
	if err != nil {
		t.Fatalf("Locate country: %v", err)
	}
	want := "https://date.nager.at/api/v3/PublicHolidays/" + currentYearStr() + "/US"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocateDate(t *testing.T) {
	got, err := Domain{}.Locate("date", "2024-01-01")
	if err != nil {
		t.Fatalf("Locate date: %v", err)
	}
	want := "https://date.nager.at/api/v3/IsTodayPublicHoliday/2024-01-01"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	got, err := h.ResolveOn("nagerdate", "US")
	if err != nil || got.String() != "nagerdate://country/US" {
		t.Errorf("ResolveOn = (%q, %v), want nagerdate://country/US", got.String(), err)
	}
}
