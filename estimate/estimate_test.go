package estimate

import (
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

func TestLoadTables(t *testing.T) {
	tables, err := LoadTables()
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	twilio, ok := tables["twilio"]
	if !ok {
		t.Fatalf("expected twilio table, got %v", tables)
	}
	if twilio.AsOf == "" || twilio.Currency != "USD" {
		t.Errorf("twilio table metadata mismatch: %+v", twilio)
	}
	if twilio.SMS["US"] != 0.0083 {
		t.Errorf("expected US 0.0083, got %v", twilio.SMS["US"])
	}
	if twilio.Verify == nil || *twilio.Verify != 0.05 {
		t.Errorf("expected verify 0.05, got %v", twilio.Verify)
	}
	termii, ok := tables["termii"]
	if !ok {
		t.Fatalf("expected termii table, got %v", tables)
	}
	if termii.SMS["NG"] != 0.005 {
		t.Errorf("expected NG 0.005, got %v", termii.SMS["NG"])
	}
}

func testMessage(provider string, to string, segments int, direction core.Direction, channel core.Channel) *core.Message {
	return &core.Message{
		ID:        core.NewMessageID(),
		ProjectID: "prj_test",
		Channel:   channel,
		Direction: direction,
		Provider:  provider,
		From:      "+15555550100",
		To:        to,
		BodyText:  "Hi",
		Status:    core.StatusDelivered,
		Segments:  segments,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestSummarize(t *testing.T) {
	messages := []*core.Message{
		testMessage("twilio", "+15005550006", 1, core.DirectionOutbound, core.ChannelSMS),
		testMessage("twilio", "+15005550007", 2, core.DirectionOutbound, core.ChannelSMS),
		testMessage("twilio", "+2348031234567", 1, core.DirectionOutbound, core.ChannelSMS),
		testMessage("twilio", "+15005550006", 1, core.DirectionInbound, core.ChannelSMS),
		testMessage("native", "+15005550006", 1, core.DirectionOutbound, core.ChannelSMS),
		testMessage("twilio", "user@example.com", 0, core.DirectionOutbound, core.ChannelEmail),
	}
	rows := Summarize(messages)
	if len(rows) != 2 {
		t.Fatalf("expected 2 mix rows, got %+v", rows)
	}
	ng := rows[0]
	if ng.Provider != "twilio" || ng.Country != "NG" || ng.Count != 1 || ng.AvgSegments != 1 {
		t.Errorf("NG row mismatch: %+v", ng)
	}
	us := rows[1]
	if us.Country != "US" || us.Count != 2 || us.AvgSegments != 1.5 {
		t.Errorf("US row mismatch: %+v", us)
	}
}

func TestEstimate(t *testing.T) {
	tables, err := LoadTables()
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	mix := []MixRow{
		{Provider: "twilio", Channel: core.ChannelSMS, Country: "US", AvgSegments: 1.5, Count: 2},
		{Provider: "twilio", Channel: core.ChannelSMS, Country: "GH", AvgSegments: 1, Count: 1},
	}

	t.Run("scales mix to volume", func(t *testing.T) {
		res := Estimate(tables["twilio"], "twilio", mix, 1, 300)
		// factor 100: 200 US + 100 GH messages, 100 verifications.
		if res.MonthlyMessages != 200 {
			t.Errorf("expected 200 priced messages, got %d", res.MonthlyMessages)
		}
		// 200 * 1.5 * 0.0083 = 2.49
		if res.TotalCost != 2.49 {
			t.Errorf("expected total 2.49, got %v", res.TotalCost)
		}
		if res.MonthlyVerifications != 100 || res.VerifyCost != 5.0 {
			t.Errorf("expected 100 verifications at 5.00, got %+v", res)
		}
		if !res.EstimateOnly || res.Currency != "USD" || res.AsOf == "" {
			t.Errorf("metadata mismatch: %+v", res)
		}
	})

	t.Run("missing countries reported not guessed", func(t *testing.T) {
		res := Estimate(tables["twilio"], "twilio", mix, 0, 300)
		if len(res.NoPriceData) != 1 || res.NoPriceData[0].Country != "GH" || res.NoPriceData[0].Messages != 100 {
			t.Errorf("expected GH missing with 100, got %+v", res.NoPriceData)
		}
	})

	t.Run("other providers ignored", func(t *testing.T) {
		res := Estimate(tables["termii"], "termii", mix, 0, 300)
		if res.MonthlyMessages != 0 || res.TotalCost != 0 {
			t.Errorf("expected zero result, got %+v", res)
		}
	})

	t.Run("zero volume zeroes out", func(t *testing.T) {
		res := Estimate(tables["twilio"], "twilio", mix, 5, 0)
		if res.MonthlyMessages != 0 || res.MonthlyVerifications != 0 || res.TotalCost != 0 {
			t.Errorf("expected zero result, got %+v", res)
		}
	})
}
