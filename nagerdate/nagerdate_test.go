package nagerdate_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/nagerdate-cli/nagerdate"
)

const fakeHolidaysJSON = `[
  {"date":"2024-01-01","localName":"New Year's Day","name":"New Year's Day","countryCode":"US","fixed":true,"global":true,"counties":null,"launchYear":null,"types":["Public"]},
  {"date":"2024-01-15","localName":"Martin Luther King, Jr. Day","name":"Martin Luther King, Jr. Day","countryCode":"US","fixed":false,"global":true,"counties":null,"launchYear":null,"types":["Public"]}
]`

const fakeCountriesJSON = `[
  {"countryCode":"AD","name":"Andorra"},
  {"countryCode":"AL","name":"Albania"},
  {"countryCode":"US","name":"United States"}
]`

const fakeLongWeekendsJSON = `[
  {"startDate":"2023-12-30","endDate":"2024-01-01","dayCount":3,"needBridgeDay":false},
  {"startDate":"2024-05-25","endDate":"2024-05-27","dayCount":3,"needBridgeDay":false}
]`

func newTestClient(baseURL string) *nagerdate.Client {
	c := nagerdate.NewClient()
	c.Rate = 0
	// Override the base URL by injecting through the config path
	return c
}

// newTestClientWithServer returns a client whose HTTP calls go to ts.
// We patch the internal base via a custom transport.
func newTestClientWithServer(ts *httptest.Server) *nagerdate.Client {
	cfg := nagerdate.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return nagerdate.NewClientFromConfig(cfg)
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := nagerdate.NewClient()
	c.Rate = 0

	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := nagerdate.NewClient()
	c.Rate = 0
	c.Retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGetHolidaysParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeHolidaysJSON)
	}))
	defer ts.Close()

	c := newTestClientWithServer(ts)
	items, err := c.GetHolidays(context.Background(), 2024, "US")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Date != "2024-01-01" {
		t.Errorf("items[0].Date = %q, want 2024-01-01", items[0].Date)
	}
	if items[0].Name != "New Year's Day" {
		t.Errorf("items[0].Name = %q, want New Year's Day", items[0].Name)
	}
	if items[0].CountryCode != "US" {
		t.Errorf("items[0].CountryCode = %q, want US", items[0].CountryCode)
	}
	if !items[0].Fixed {
		t.Error("items[0].Fixed should be true")
	}
}

func TestGetHolidaysSendsUA(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeHolidaysJSON)
	}))
	defer ts.Close()

	c := newTestClientWithServer(ts)
	_, err := c.GetHolidays(context.Background(), 2024, "US")
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestGetCountriesParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeCountriesJSON)
	}))
	defer ts.Close()

	c := newTestClientWithServer(ts)
	items, err := c.GetCountries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0].CountryCode != "AD" {
		t.Errorf("items[0].CountryCode = %q, want AD", items[0].CountryCode)
	}
	if items[2].CountryCode != "US" {
		t.Errorf("items[2].CountryCode = %q, want US", items[2].CountryCode)
	}
	if items[2].Name != "United States" {
		t.Errorf("items[2].Name = %q, want United States", items[2].Name)
	}
}

func TestGetNextWorldwideParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeHolidaysJSON)
	}))
	defer ts.Close()

	c := newTestClientWithServer(ts)
	items, err := c.GetNextWorldwide(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestGetLongWeekendsParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeLongWeekendsJSON)
	}))
	defer ts.Close()

	c := newTestClientWithServer(ts)
	items, err := c.GetLongWeekends(context.Background(), 2024, "US")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].StartDate != "2023-12-30" {
		t.Errorf("items[0].StartDate = %q, want 2023-12-30", items[0].StartDate)
	}
	if items[0].DayCount != 3 {
		t.Errorf("items[0].DayCount = %d, want 3", items[0].DayCount)
	}
	if items[0].NeedBridgeDay {
		t.Error("items[0].NeedBridgeDay should be false")
	}
}
