package nagerdate

// Holiday is one public holiday record.
type Holiday struct {
	Date        string   `kit:"id" json:"date"`
	LocalName   string   `json:"local_name"`
	Name        string   `json:"name"`
	CountryCode string   `json:"country_code"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties"`
	Types       []string `json:"types"`
}

// Country is one supported country.
type Country struct {
	CountryCode string `kit:"id" json:"country_code"`
	Name        string `json:"name"`
}

// LongWeekend is one long weekend record.
type LongWeekend struct {
	StartDate     string `kit:"id" json:"start_date"`
	EndDate       string `json:"end_date"`
	DayCount      int    `json:"day_count"`
	NeedBridgeDay bool   `json:"need_bridge_day"`
}

// wire types — only used inside nagerdate.go for JSON decode

type wireHoliday struct {
	Date        string   `json:"date"`
	LocalName   string   `json:"localName"`
	Name        string   `json:"name"`
	CountryCode string   `json:"countryCode"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties"`
	LaunchYear  *int     `json:"launchYear"`
	Types       []string `json:"types"`
}

type wireCountry struct {
	CountryCode string `json:"countryCode"`
	Name        string `json:"name"`
}

type wireLongWeekend struct {
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	DayCount      int    `json:"dayCount"`
	NeedBridgeDay bool   `json:"needBridgeDay"`
}
