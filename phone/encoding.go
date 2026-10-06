package phone

import "unicode/utf16"

type Encoding int

const (
	EncodingGSM7 Encoding = iota
	EncodingUCS2
)

func (e Encoding) String() string {
	switch e {
	case EncodingGSM7:
		return "gsm7"
	case EncodingUCS2:
		return "ucs2"
	default:
		return "unknown"
	}
}

const (
	gsm7SingleMax = 160
	gsm7MultiMax  = 153
	ucs2SingleMax = 70
	ucs2MultiMax  = 67
	maxSegments   = 10
)

var gsm7Basic = map[rune]bool{
	'@': true, '£': true, '$': true, '¥': true, 'è': true, 'é': true, 'ù': true, 'ì': true, 'ò': true, 'Ç': true,
	'\n': true, 'Ø': true, 'ø': true, '\r': true, 'Å': true, 'å': true, 'Δ': true, '_': true, 'Φ': true, 'Γ': true,
	'Λ': true, 'Ω': true, 'Π': true, 'Ψ': true, 'Σ': true, 'Θ': true, 'Ξ': true, 'Æ': true, 'æ': true, 'ß': true, 'É': true,
	' ': true, '!': true, '"': true, '#': true, '¤': true, '%': true, '&': true, '\'': true, '(': true, ')': true,
	'*': true, '+': true, ',': true, '-': true, '.': true, '/': true, '0': true, '1': true, '2': true, '3': true,
	'4': true, '5': true, '6': true, '7': true, '8': true, '9': true, ':': true, ';': true, '<': true, '=': true,
	'>': true, '?': true, '¡': true, 'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true,
	'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true, 'Q': true,
	'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true, 'Ä': true,
	'Ö': true, 'Ñ': true, 'Ü': true, '§': true, '¿': true, 'a': true, 'b': true, 'c': true, 'd': true, 'e': true,
	'f': true, 'g': true, 'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true,
	'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true, 'y': true,
	'z': true, 'ä': true, 'ö': true, 'ñ': true, 'ü': true, 'à': true,
}

var gsm7Extended = map[rune]bool{
	'^': true, '{': true, '}': true, '\\': true, '[': true, ']': true, '~': true, '|': true, '€': true, '\f': true,
}

func Analyze(body string) (Encoding, int) {
	if body == "" {
		return EncodingGSM7, 0
	}

	encoding := EncodingGSM7
	gsm7Len := 0

	for _, r := range body {
		if gsm7Basic[r] {
			gsm7Len++
		} else if gsm7Extended[r] {
			gsm7Len += 2
			encoding = EncodingGSM7
		} else {
			return EncodingUCS2, countSegmentsUCS2(body)
		}
	}

	return encoding, countSegmentsGSM7(gsm7Len)
}

func countSegmentsGSM7(length int) int {
	if length == 0 {
		return 0
	}
	if length <= gsm7SingleMax {
		return 1
	}
	// First segment: 160 chars, subsequent: 153 chars each
	remaining := length - gsm7SingleMax
	segs := 1 + (remaining+gsm7MultiMax-1)/gsm7MultiMax
	return segs
}

func countSegmentsUCS2(body string) int {
	length := 0
	for _, r := range body {
		length += utf16.RuneLen(r)
	}
	if length == 0 {
		return 0
	}
	if length <= ucs2SingleMax {
		return 1
	}
	// First segment: 70 chars, subsequent: 67 chars each
	remaining := length - ucs2SingleMax
	segs := 1 + (remaining+ucs2MultiMax-1)/ucs2MultiMax
	return segs
}

func MaxSegments(encoding Encoding) int {
	return maxSegments
}

func SegmentCapacity(encoding Encoding, segment int) int {
	if segment == 1 {
		if encoding == EncodingGSM7 {
			return gsm7SingleMax
		}
		return ucs2SingleMax
	}
	if encoding == EncodingGSM7 {
		return gsm7MultiMax
	}
	return ucs2MultiMax
}

func IsGSM7(r rune) bool {
	return gsm7Basic[r] || gsm7Extended[r]
}

func GSM7CharCount(body string) int {
	count := 0
	for _, r := range body {
		if gsm7Basic[r] {
			count++
		} else if gsm7Extended[r] {
			count += 2
		} else {
			return -1
		}
	}
	return count
}

func UCS2CharCount(body string) int {
	return len([]rune(body))
}
