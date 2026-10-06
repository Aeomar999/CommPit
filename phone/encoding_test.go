package phone

import (
	"strings"
	"testing"
)

func TestEncoding(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantEnc  Encoding
		wantSegs int
	}{
		{
			name:     "pure GSM-7 single segment",
			body:     "Hello world",
			wantEnc:  EncodingGSM7,
			wantSegs: 1,
		},
		{
			name:     "GSM-7 exactly 160 chars",
			body:     strings.Repeat("a", 160),
			wantEnc:  EncodingGSM7,
			wantSegs: 1,
		},
		{
			name:     "GSM-7 161 chars = 2 segments",
			body:     strings.Repeat("a", 161),
			wantEnc:  EncodingGSM7,
			wantSegs: 2,
		},
		{
			name:     "GSM-7 306 chars = 2 segments (153 per segment)",
			body:     strings.Repeat("a", 306),
			wantEnc:  EncodingGSM7,
			wantSegs: 2,
		},
		{
			name:     "GSM-7 307 chars = 2 segments (160 + 147)",
			body:     strings.Repeat("a", 307),
			wantEnc:  EncodingGSM7,
			wantSegs: 2,
		},
		{
			name:     "UCS-2 single char",
			body:     "😀",
			wantEnc:  EncodingUCS2,
			wantSegs: 1,
		},
		{
			name:     "UCS-2 exactly 70 chars",
			body:     strings.Repeat("あ", 70),
			wantEnc:  EncodingUCS2,
			wantSegs: 1,
		},
		{
			name:     "UCS-2 71 chars = 2 segments",
			body:     strings.Repeat("あ", 71),
			wantEnc:  EncodingUCS2,
			wantSegs: 2,
		},
		{
			name:     "UCS-2 134 chars = 2 segments (67 per segment)",
			body:     strings.Repeat("あ", 134),
			wantEnc:  EncodingUCS2,
			wantSegs: 2,
		},
		{
			name:     "UCS-2 135 chars = 2 segments (70 + 65)",
			body:     strings.Repeat("あ", 135),
			wantEnc:  EncodingUCS2,
			wantSegs: 2,
		},
		{
			name:     "mixed GSM-7 and UCS-2 forces UCS-2",
			body:     "Hello 😀 world",
			wantEnc:  EncodingUCS2,
			wantSegs: 1,
		},
		{
			name:     "GSM-7 extended chars (^ { } [ ] ~ | €)",
			body:     "^{}[~|€]",
			wantEnc:  EncodingGSM7,
			wantSegs: 1,
		},
		{
			name:     "GSM-7 extended char counts as 2",
			body:     strings.Repeat("€", 78) + strings.Repeat("a", 4), // 78*2 + 4 = 160
			wantEnc:  EncodingGSM7,
			wantSegs: 1,
		},
		{
			name:     "GSM-7 extended char overflow",
			body:     strings.Repeat("€", 79), // 79*2 = 158, but next would be 160+
			wantEnc:  EncodingGSM7,
			wantSegs: 1,
		},
		{
			name:     "empty message",
			body:     "",
			wantEnc:  EncodingGSM7,
			wantSegs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, segs := Analyze(tt.body)
			if enc != tt.wantEnc {
				t.Errorf("expected encoding %v, got %v", tt.wantEnc, enc)
			}
			if segs != tt.wantSegs {
				t.Errorf("expected %d segments, got %d", tt.wantSegs, segs)
			}
		})
	}
}

func TestGSM7Charset(t *testing.T) {
	// All GSM-7 basic charset characters
	basic := "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	enc, segs := Analyze(basic)
	if enc != EncodingGSM7 {
		t.Errorf("basic GSM-7 charset should be GSM-7, got %v", enc)
	}
	if segs != 1 {
		t.Errorf("basic GSM-7 charset should be 1 segment, got %d", segs)
	}

	// Extended charset (requires escape, counts as 2 chars each)
	extended := "^{}\\[]~|€"
	enc, segs = Analyze(extended)
	if enc != EncodingGSM7 {
		t.Errorf("extended GSM-7 charset should be GSM-7, got %v", enc)
	}
	if segs != 1 {
		t.Errorf("extended GSM-7 charset should be 1 segment, got %d", segs)
	}
}

func TestMaxSegments(t *testing.T) {
	// Maximum 10 segments for concatenated SMS - Analyze returns uncapped count
	// 10 segments = 160 + 9*153 = 1537 chars
	longGSM7 := strings.Repeat("a", 1537)
	_, segs := Analyze(longGSM7)
	if segs != 10 {
		t.Errorf("expected 10 segments for 1537 GSM-7 chars, got %d", segs)
	}

	// 10 segments = 70 + 9*67 = 673 UCS-2 chars (but each あ is 2 UTF-16 units)
	// 673 runes * 2 = 1346 UTF-16 units
	longUCS2 := strings.Repeat("あ", 673)
	_, segs = Analyze(longUCS2)
	if segs != 10 {
		t.Errorf("expected 10 segments for 673 UCS-2 chars, got %d", segs)
	}

	// 11th segment - returns 11 (uncapped)
	tooLongGSM7 := strings.Repeat("a", 1538)
	_, segs = Analyze(tooLongGSM7)
	if segs != 11 {
		t.Errorf("expected 11 segments for 1538 GSM-7 chars (uncapped), got %d", segs)
	}
}
