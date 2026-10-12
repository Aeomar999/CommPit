// Package estimate turns observed sandbox traffic plus a developer-entered
// monthly volume into per-provider go-live cost estimates (spec §8.6).
//
// Prices come from the embedded YAML tables with an as_of date; countries
// without data are reported, never guessed. Email channels and the
// sandbox-local native/smtp providers are out of scope (Wave 2 for
// email) and excluded from the mix.
package estimate

import (
	"embed"
	"fmt"
	"math"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/phone"
)

//go:embed pricing/*.yaml
var pricingFS embed.FS

// Table is one provider's parsed pricing file.
type Table struct {
	Provider      string             `yaml:"provider"`
	Currency      string             `yaml:"currency"`
	AsOf          string             `yaml:"as_of"`
	Source        string             `yaml:"source"`
	SMS           map[string]float64 `yaml:"sms"`
	Verify        *float64           `yaml:"verify"`
	DNDMultiplier *float64           `yaml:"dnd_multiplier"`
}

// LoadTables parses every embedded pricing file, keyed by provider.
func LoadTables() (map[string]*Table, error) {
	entries, err := pricingFS.ReadDir("pricing")
	if err != nil {
		return nil, fmt.Errorf("estimate: list pricing: %w", err)
	}
	tables := make(map[string]*Table, len(entries))
	for _, entry := range entries {
		raw, err := pricingFS.ReadFile("pricing/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("estimate: read %s: %w", entry.Name(), err)
		}
		var table Table
		if err := yaml.Unmarshal(raw, &table); err != nil {
			return nil, fmt.Errorf("estimate: parse %s: %w", entry.Name(), err)
		}
		if table.Provider == "" {
			return nil, fmt.Errorf("estimate: %s has no provider", entry.Name())
		}
		tables[table.Provider] = &table
	}
	return tables, nil
}

// MixRow is one observed traffic group.
type MixRow struct {
	Provider    string
	Channel     core.Channel
	Country     string
	AvgSegments float64
	Count       int
}

// Summarize folds outbound SMS messages into mix rows grouped by
// provider, channel and destination country, with average segments.
func Summarize(messages []*core.Message) []MixRow {
	type key struct {
		provider string
		channel  core.Channel
		country  string
	}
	counts := map[key]int{}
	segments := map[key]int{}
	for _, msg := range messages {
		if msg == nil || msg.Direction != core.DirectionOutbound || msg.Channel != core.ChannelSMS {
			continue
		}
		if msg.Provider != "twilio" && msg.Provider != "termii" {
			continue
		}
		country := ""
		if parsed, err := phone.Parse(msg.To, phone.ModeValid); err == nil {
			country = parsed.Country
		}
		k := key{msg.Provider, msg.Channel, country}
		counts[k]++
		segments[k] += msg.Segments
	}
	rows := make([]MixRow, 0, len(counts))
	for k, count := range counts {
		rows = append(rows, MixRow{
			Provider:    k.provider,
			Channel:     k.channel,
			Country:     k.country,
			AvgSegments: float64(segments[k]) / float64(count),
			Count:       count,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Provider != rows[j].Provider {
			return rows[i].Provider < rows[j].Provider
		}
		if rows[i].Country != rows[j].Country {
			return rows[i].Country < rows[j].Country
		}
		return rows[i].Channel < rows[j].Channel
	})
	return rows
}

// CountryCost is one priced destination row.
type CountryCost struct {
	Country   string  `json:"country"`
	Messages  int     `json:"messages"`
	UnitPrice float64 `json:"unit_price"`
	Cost      float64 `json:"cost"`
}

// MissingPrice is one destination without table data.
type MissingPrice struct {
	Country  string `json:"country"`
	Messages int    `json:"messages"`
}

// Result is one provider's monthly estimate.
type Result struct {
	Provider             string         `json:"provider"`
	Currency             string         `json:"currency"`
	AsOf                 string         `json:"as_of"`
	EstimateOnly         bool           `json:"estimate_only"`
	MonthlyMessages      int            `json:"monthly_messages"`
	TotalCost            float64        `json:"total_cost"`
	ByCountry            []CountryCost  `json:"by_country"`
	MonthlyVerifications int            `json:"monthly_verifications"`
	VerifyCost           float64        `json:"verify_cost"`
	NoPriceData          []MissingPrice `json:"no_price_data"`
}

func cents(v float64) float64 { return math.Round(v*100) / 100 }

// Estimate scales one provider's observed mix to volume messages per
// month. verifications is the observed verification count, scaled by the
// same factor. Zero volume or an empty mix yields a zeroed result.
func Estimate(table *Table, provider string, mix []MixRow, verifications, volume int) Result {
	res := Result{
		Provider:     provider,
		Currency:     table.Currency,
		AsOf:         table.AsOf,
		EstimateOnly: true,
		ByCountry:    []CountryCost{},
		NoPriceData:  []MissingPrice{},
	}
	total := 0
	for _, row := range mix {
		if row.Provider == provider {
			total += row.Count
		}
	}
	if total == 0 || volume <= 0 {
		return res
	}
	factor := float64(volume) / float64(total)
	for _, row := range mix {
		if row.Provider != provider {
			continue
		}
		monthly := int(math.Round(float64(row.Count) * factor))
		unit, ok := table.SMS[row.Country]
		if !ok {
			res.NoPriceData = append(res.NoPriceData, MissingPrice{Country: row.Country, Messages: monthly})
			continue
		}
		cost := cents(float64(monthly) * row.AvgSegments * unit)
		res.ByCountry = append(res.ByCountry, CountryCost{
			Country: row.Country, Messages: monthly, UnitPrice: unit, Cost: cost,
		})
		res.MonthlyMessages += monthly
		res.TotalCost += cost
	}
	res.TotalCost = cents(res.TotalCost)
	res.MonthlyVerifications = int(math.Round(float64(verifications) * factor))
	if table.Verify != nil {
		res.VerifyCost = cents(float64(res.MonthlyVerifications) * *table.Verify)
	}
	sort.Slice(res.ByCountry, func(i, j int) bool { return res.ByCountry[i].Country < res.ByCountry[j].Country })
	sort.Slice(res.NoPriceData, func(i, j int) bool { return res.NoPriceData[i].Country < res.NoPriceData[j].Country })
	return res
}
