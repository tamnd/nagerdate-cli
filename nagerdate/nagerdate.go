// Package nagerdate is the library behind the nagerdate command line:
// the HTTP client, request shaping, and the typed data models for the
// Nager.Date public holiday API (date.nager.at).
//
// No API key or authentication is required. The API provides public holiday
// data for 123 countries.
package nagerdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultUserAgent identifies the client to the server.
const DefaultUserAgent = "nagerdate/dev (+https://github.com/tamnd/nagerdate-cli)"

// Host is the site this client talks to.
const Host = "date.nager.at"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults for the Nager.Date API.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://date.nager.at",
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Timeout:   15 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Nager.Date API over HTTP.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	base string
	last time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	cfg := DefaultConfig()
	return &Client{
		HTTP:      &http.Client{Timeout: cfg.Timeout},
		UserAgent: cfg.UserAgent,
		Rate:      cfg.Rate,
		Retries:   cfg.Retries,
	}
}

// NewClientFromConfig builds a Client from an explicit Config.
func NewClientFromConfig(cfg Config) *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: cfg.Timeout},
		UserAgent: cfg.UserAgent,
		Rate:      cfg.Rate,
		Retries:   cfg.Retries,
		base:      cfg.BaseURL,
	}
}

// baseURL returns the configured base URL, falling back to the default.
func (c *Client) baseURL() string {
	if c.base != "" {
		return c.base
	}
	return "https://" + Host
}

// GetHolidays fetches public holidays for the given country and year.
func (c *Client) GetHolidays(ctx context.Context, year int, countryCode string) ([]Holiday, error) {
	u := fmt.Sprintf("%s/api/v3/PublicHolidays/%d/%s", c.baseURL(), year, countryCode)
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []wireHoliday
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode holidays: %w", err)
	}
	items := make([]Holiday, 0, len(raw))
	for _, r := range raw {
		items = append(items, Holiday{
			Date:        r.Date,
			LocalName:   r.LocalName,
			Name:        r.Name,
			CountryCode: r.CountryCode,
			Fixed:       r.Fixed,
			Global:      r.Global,
			Counties:    r.Counties,
			Types:       r.Types,
		})
	}
	return items, nil
}

// GetCountries returns all available countries supported by the API.
func (c *Client) GetCountries(ctx context.Context) ([]Country, error) {
	u := fmt.Sprintf("%s/api/v3/AvailableCountries", c.baseURL())
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []wireCountry
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode countries: %w", err)
	}
	items := make([]Country, 0, len(raw))
	for _, r := range raw {
		items = append(items, Country{
			CountryCode: r.CountryCode,
			Name:        r.Name,
		})
	}
	return items, nil
}

// GetNextWorldwide fetches the next public holidays worldwide.
func (c *Client) GetNextWorldwide(ctx context.Context) ([]Holiday, error) {
	u := fmt.Sprintf("%s/api/v3/NextPublicHolidaysWorldwide", c.baseURL())
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []wireHoliday
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode next worldwide: %w", err)
	}
	items := make([]Holiday, 0, len(raw))
	for _, r := range raw {
		items = append(items, Holiday{
			Date:        r.Date,
			LocalName:   r.LocalName,
			Name:        r.Name,
			CountryCode: r.CountryCode,
			Fixed:       r.Fixed,
			Global:      r.Global,
			Counties:    r.Counties,
			Types:       r.Types,
		})
	}
	return items, nil
}

// GetLongWeekends fetches long weekends for the given country and year.
func (c *Client) GetLongWeekends(ctx context.Context, year int, countryCode string) ([]LongWeekend, error) {
	u := fmt.Sprintf("%s/api/v3/LongWeekend/%d/%s", c.baseURL(), year, countryCode)
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []wireLongWeekend
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode long weekends: %w", err)
	}
	items := make([]LongWeekend, 0, len(raw))
	for _, r := range raw {
		items = append(items, LongWeekend{
			StartDate:     r.StartDate,
			EndDate:       r.EndDate,
			DayCount:      r.DayCount,
			NeedBridgeDay: r.NeedBridgeDay,
		})
	}
	return items, nil
}

// Get fetches url and returns the response body. It paces and retries according
// to the client's settings.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
