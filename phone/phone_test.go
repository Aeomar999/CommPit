package phone

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		mode    Mode
		want    *Parsed
		wantErr bool
		errCode string
	}{
		{
			name:  "valid E.164 US",
			input: "+14155552671",
			mode:  ModeValid,
			want: &Parsed{
				E164:     "+14155552671",
				Country:  "US",
				National: "4155552671",
				Valid:    true,
				Possible: true,
			},
			wantErr: false,
		},
		{
			name:  "valid E.164 UK with spaces",
			input: "+44 20 7123 4567",
			mode:  ModeValid,
			want: &Parsed{
				E164:     "+442071234567",
				Country:  "GB",
				National: "02071234567",
				Valid:    true,
				Possible: true,
			},
			wantErr: false,
		},
		{
			name:  "valid E.164 US with parentheses",
			input: "+1 (415) 555-2671",
			mode:  ModeValid,
			want: &Parsed{
				E164:     "+14155552671",
				Country:  "US",
				National: "4155552671",
				Valid:    true,
				Possible: true,
			},
			wantErr: false,
		},
		{
			name:    "invalid number in valid mode",
			input:   "not-a-number",
			mode:    ModeValid,
			want:    nil,
			wantErr: true,
			errCode: "invalid_number",
		},
		{
			name:    "impossible number in valid mode",
			input:   "+123",
			mode:    ModeValid,
			want:    nil,
			wantErr: true,
			errCode: "invalid_number",
		},
		{
			name:  "possible but not valid in possible mode",
			input: "+15550000000",
			mode:  ModePossible,
			want: &Parsed{
				E164:     "+15550000000",
				Country:  "",
				National: "5550000000",
				Valid:    false,
				Possible: true,
			},
			wantErr: false,
		},
		{
			name:    "impossible number in possible mode",
			input:   "+123",
			mode:    ModePossible,
			want:    nil,
			wantErr: true,
			errCode: "invalid_number",
		},
		{
			name:  "any string in off mode",
			input: "anything",
			mode:  ModeOff,
			want: &Parsed{
				E164:     "anything",
				Country:  "",
				National: "",
				Valid:    false,
				Possible: false,
			},
			wantErr: false,
		},
		{
			name:  "empty string in off mode",
			input: "",
			mode:  ModeOff,
			want: &Parsed{
				E164:     "",
				Country:  "",
				National: "",
				Valid:    false,
				Possible: false,
			},
			wantErr: false,
		},
		{
			name:    "empty string in valid mode",
			input:   "",
			mode:    ModeValid,
			want:    nil,
			wantErr: true,
			errCode: "validation_error",
		},
		{
			name:    "missing plus sign in valid mode",
			input:   "14155552671",
			mode:    ModeValid,
			want:    nil,
			wantErr: true,
			errCode: "invalid_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, tt.mode)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var pe *Error
				if !errors.As(err, &pe) {
					t.Fatalf("expected *Error, got %T", err)
				}
				if pe.Code != tt.errCode {
					t.Fatalf("expected error code %q, got %q", tt.errCode, pe.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParse_CountryCode(t *testing.T) {
	tests := []struct {
		input   string
		country string
	}{
		{"+14155552671", "US"},
		{"+442071234567", "GB"},
		{"+33123456789", "FR"},
		{"+493012345678", "DE"},
		{"+61234567890", "AU"},
		{"+81312345678", "JP"},
		{"+8613800138000", "CN"},
		{"+919876543210", "IN"},
		{"+5511999999999", "BR"},
		{"+27821234567", "ZA"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsed, err := Parse(tt.input, ModeValid)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Country != tt.country {
				t.Errorf("expected country %q, got %q", tt.country, parsed.Country)
			}
		})
	}
}
