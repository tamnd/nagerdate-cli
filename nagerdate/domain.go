package nagerdate

import (
	"context"
	"regexp"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes nagerdate as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/nagerdate-cli/nagerdate"
//
// The same Domain also builds the standalone nagerdate binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the nagerdate driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "nagerdate",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "nagerdate",
			Short:  "Public holiday CLI for 123 countries (date.nager.at).",
			Long: `nagerdate fetches public holiday data from date.nager.at.
No API key required. Covers 123 countries.`,
			Site: Host,
			Repo: "https://github.com/tamnd/nagerdate-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// holidays: list public holidays for a country and year
	kit.Handle(app, kit.OpMeta{
		Name:    "holidays",
		Group:   "read",
		List:    true,
		Summary: "List public holidays for a country and year",
	}, holidaysOp)

	// countries: list all supported countries
	kit.Handle(app, kit.OpMeta{
		Name:    "countries",
		Group:   "read",
		List:    true,
		Summary: "List all 123 countries supported by the API",
	}, countriesOp)

	// next: next public holidays worldwide
	kit.Handle(app, kit.OpMeta{
		Name:    "next",
		Group:   "read",
		List:    true,
		Summary: "List next public holidays worldwide",
	}, nextOp)

	// longweekend: long weekends for a country and year
	kit.Handle(app, kit.OpMeta{
		Name:    "longweekend",
		Group:   "read",
		List:    true,
		Summary: "List long weekends for a country and year",
	}, longweekendOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClientFromConfig(c), nil
}

// --- inputs ---

type holidaysInput struct {
	Year    int           `kit:"flag" help:"year (e.g. 2024)" default:"0"`
	Country string        `kit:"flag" help:"country code (e.g. US, GB, FR)" default:"US"`
	Limit   int           `kit:"flag,inherit" help:"max results"`
	Delay   time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client  *Client       `kit:"inject"`
}

type countriesInput struct {
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type nextInput struct {
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type longweekendInput struct {
	Year    int           `kit:"flag" help:"year (e.g. 2024)" default:"0"`
	Country string        `kit:"flag" help:"country code (e.g. US, GB, FR)" default:"US"`
	Limit   int           `kit:"flag,inherit" help:"max results"`
	Delay   time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client  *Client       `kit:"inject"`
}

// --- handlers ---

func holidaysOp(ctx context.Context, in holidaysInput, emit func(Holiday) error) error {
	year := in.Year
	if year <= 0 {
		year = currentYear()
	}
	items, err := in.Client.GetHolidays(ctx, year, in.Country)
	if err != nil {
		return mapErr(err)
	}
	for i, item := range items {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func countriesOp(ctx context.Context, in countriesInput, emit func(Country) error) error {
	items, err := in.Client.GetCountries(ctx)
	if err != nil {
		return mapErr(err)
	}
	for i, item := range items {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func nextOp(ctx context.Context, in nextInput, emit func(Holiday) error) error {
	items, err := in.Client.GetNextWorldwide(ctx)
	if err != nil {
		return mapErr(err)
	}
	for i, item := range items {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func longweekendOp(ctx context.Context, in longweekendInput, emit func(LongWeekend) error) error {
	year := in.Year
	if year <= 0 {
		year = currentYear()
	}
	items, err := in.Client.GetLongWeekends(ctx, year, in.Country)
	if err != nil {
		return mapErr(err)
	}
	for i, item := range items {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

var (
	countryRE = regexp.MustCompile(`^[A-Z]{2}$`)
	dateRE    = regexp.MustCompile(`^\d{4}(-\d{2}-\d{2})?$`)
)

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty nagerdate reference")
	}
	if countryRE.MatchString(input) {
		return "country", input, nil
	}
	if dateRE.MatchString(input) {
		return "date", input, nil
	}
	return "query", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "country":
		return "https://date.nager.at/api/v3/PublicHolidays/" + currentYearStr() + "/" + id, nil
	case "date":
		return "https://date.nager.at/api/v3/IsTodayPublicHoliday/" + id, nil
	default:
		return "", errs.Usage("nagerdate has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
