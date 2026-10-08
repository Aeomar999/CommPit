package extract

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractCodes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "simple 6-digit code",
			body: "Your code is 123456",
			want: []string{"123456"},
		},
		{
			name: "4-digit code",
			body: "Code: 1234",
			want: []string{"1234"},
		},
		{
			name: "8-digit code",
			body: "Your verification code is 12345678",
			want: []string{"12345678"},
		},
		{
			name: "code with keyword 'code'",
			body: "Your code is 482913",
			want: []string{"482913"},
		},
		{
			name: "code with keyword 'otp'",
			body: "OTP: 555666",
			want: []string{"555666"},
		},
		{
			name: "code with keyword 'pin'",
			body: "Your PIN is 777888",
			want: []string{"777888"},
		},
		{
			name: "code with keyword 'verification'",
			body: "Verification code: 999000",
			want: []string{"999000"},
		},
		{
			name: "code with keyword 'token'",
			body: "Your token: 111222",
			want: []string{"111222"},
		},
		{
			name: "code with keyword 'passcode'",
			body: "Passcode: 333444",
			want: []string{"333444"},
		},
		{
			name: "multiple codes - keyword adjacent first",
			body: "Your code is 123456 and also 789012",
			want: []string{"123456", "789012"},
		},
		{
			name: "code not adjacent to keyword comes after",
			body: "Here is a number 111111 and your code is 222222",
			want: []string{"222222", "111111"},
		},
		{
			name: "alphanumeric code adjacent to keyword",
			body: "Your code is AB12CD34",
			want: []string{"AB12CD34"},
		},
		{
			name: "alphanumeric code not adjacent to keyword",
			body: "Random AB12CD34 and code is EF56GH78",
			want: []string{"EF56GH78", "AB12CD34"},
		},
		{
			name: "no codes found",
			body: "Hello world",
			want: []string{},
		},
		{
			name: "code with punctuation",
			body: "Code: 123-456",
			want: []string{},
		},
		{
			name: "3-digit too short",
			body: "Code: 123",
			want: []string{},
		},
		{
			name: "9-digit too long",
			body: "Code: 123456789",
			want: []string{},
		},
		{
			name: "code in HTML",
			body: "<p>Your code is <strong>123456</strong></p>",
			want: []string{"123456"},
		},
		{
			name: "case insensitive keywords",
			body: "Your CODE is 123456",
			want: []string{"123456"},
		},
		{
			name: "keyword at start of message",
			body: "OTP 123456 is your code",
			want: []string{"123456"},
		},
		{
			name: "keyword at end of message",
			body: "Your code is 123456 OTP",
			want: []string{"123456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCodes(tt.body)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExtractCodes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "simple http link",
			body: "Check https://example.com",
			want: []string{"https://example.com"},
		},
		{
			name: "simple https link",
			body: "Visit http://example.com",
			want: []string{"http://example.com"},
		},
		{
			name: "multiple links",
			body: "Links: https://a.com and https://b.com",
			want: []string{"https://a.com", "https://b.com"},
		},
		{
			name: "link with path and query",
			body: "Go to https://example.com/verify?token=abc123",
			want: []string{"https://example.com/verify?token=abc123"},
		},
		{
			name: "link in HTML href",
			body: `<a href="https://example.com/verify">Click here</a>`,
			want: []string{"https://example.com/verify"},
		},
		{
			name: "multiple links in HTML",
			body: `<a href="https://a.com">A</a> and <a href="https://b.com">B</a>`,
			want: []string{"https://a.com", "https://b.com"},
		},
		{
			name: "mixed text and HTML links",
			body: `Text https://text.com and <a href="https://html.com">HTML</a>`,
			want: []string{"https://text.com", "https://html.com"},
		},
		{
			name: "no links",
			body: "Hello world",
			want: []string{},
		},
		{
			name: "link with port",
			body: "http://localhost:8080/verify",
			want: []string{"http://localhost:8080/verify"},
		},
		{
			name: "link with fragment",
			body: "https://example.com#section",
			want: []string{"https://example.com#section"},
		},
		{
			name: "deduplicated links",
			body: "https://example.com and https://example.com",
			want: []string{"https://example.com"},
		},
		{
			name: "link with subdomain",
			body: "https://verify.example.com/path",
			want: []string{"https://verify.example.com/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLinks(tt.body)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExtractLinks() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPrimaryLink(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "verify keyword in URL",
			body: "https://example.com/verify?code=123",
			want: "https://example.com/verify?code=123",
		},
		{
			name: "confirm keyword in URL",
			body: "https://example.com/confirm?token=abc",
			want: "https://example.com/confirm?token=abc",
		},
		{
			name: "activate keyword in URL",
			body: "https://example.com/activate/123",
			want: "https://example.com/activate/123",
		},
		{
			name: "magic keyword in URL",
			body: "https://example.com/magic-link",
			want: "https://example.com/magic-link",
		},
		{
			name: "token keyword in URL",
			body: "https://example.com/token/abc",
			want: "https://example.com/token/abc",
		},
		{
			name: "reset keyword in URL",
			body: "https://example.com/reset-password",
			want: "https://example.com/reset-password",
		},
		{
			name: "login keyword in URL",
			body: "https://example.com/login",
			want: "https://example.com/login",
		},
		{
			name: "signin keyword in URL",
			body: "https://example.com/signin",
			want: "https://example.com/signin",
		},
		{
			name: "verify keyword in anchor text",
			body: `<a href="https://example.com/xyz">Verify your email</a>`,
			want: "https://example.com/xyz",
		},
		{
			name: "confirm keyword in anchor text",
			body: `<a href="https://example.com/abc">Confirm account</a>`,
			want: "https://example.com/abc",
		},
		{
			name: "first link when no keywords match",
			body: "https://example.com/first and https://example.com/second",
			want: "https://example.com/first",
		},
		{
			name: "no links",
			body: "Hello world",
			want: "",
		},
		{
			name: "keyword in URL path takes priority over later verify link",
			body: "https://example.com/first and https://example.com/verify",
			want: "https://example.com/verify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PrimaryLink(tt.body)
			if got != tt.want {
				t.Errorf("PrimaryLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractAll(t *testing.T) {
	body := `Your verification code is 482913. 
Please verify at https://example.com/verify?token=abc123
Or visit https://example.com/dashboard`

	result := ExtractAll(body)

	if len(result.Codes) != 1 || result.Codes[0] != "482913" {
		t.Errorf("expected codes [482913], got %v", result.Codes)
	}
	if len(result.Links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(result.Links), result.Links)
	}
	if result.PrimaryLink != "https://example.com/verify?token=abc123" {
		t.Errorf("expected primary link verify, got %q", result.PrimaryLink)
	}
}

func TestExtractAll_HTML(t *testing.T) {
	body := `<html>
<body>
<p>Your code is <strong>123456</strong></p>
<p><a href="https://example.com/verify">Verify email</a></p>
<p><a href="https://example.com/other">Other link</a></p>
</body>
</html>`

	result := ExtractAll(body)

	if len(result.Codes) != 1 || result.Codes[0] != "123456" {
		t.Errorf("expected codes [123456], got %v", result.Codes)
	}
	if len(result.Links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(result.Links), result.Links)
	}
	if result.PrimaryLink != "https://example.com/verify" {
		t.Errorf("expected primary link verify, got %q", result.PrimaryLink)
	}
}
