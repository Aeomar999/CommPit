package extract

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	codeKeywords = []string{"code", "otp", "pin", "verification", "token", "passcode"}

	digitCodeRe     = regexp.MustCompile(`\b\d{4,8}\b`)
	alphanumericRe  = regexp.MustCompile(`\b[A-Z0-9]{4,8}\b`)
	urlRe           = regexp.MustCompile(`https?://[^\s<>"']+`)
	primaryKeywords = []string{"verify", "confirm", "activate", "magic", "token", "reset", "login", "signin"}
)

func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

type Result struct {
	Codes       []string
	Links       []string
	PrimaryLink string
}

func ExtractCodes(body string) []string {
	var keywordCodes []string
	var otherCodes []string

	bodyLower := strings.ToLower(body)

	digitMatches := digitCodeRe.FindAllStringIndex(body, -1)
	for _, match := range digitMatches {
		code := body[match[0]:match[1]]
		if isNearKeyword(bodyLower, match[0]) {
			keywordCodes = append(keywordCodes, code)
		} else {
			otherCodes = append(otherCodes, code)
		}
	}

	alphaMatches := alphanumericRe.FindAllStringIndex(body, -1)
	for _, match := range alphaMatches {
		code := body[match[0]:match[1]]
		if !hasDigit(code) {
			continue
		}
		if isNearKeyword(bodyLower, match[0]) {
			keywordCodes = append(keywordCodes, code)
		} else {
			otherCodes = append(otherCodes, code)
		}
	}

	result := make([]string, 0, len(keywordCodes)+len(otherCodes))
	result = append(result, keywordCodes...)
	result = append(result, otherCodes...)

	return deduplicate(result)
}

func isNearKeyword(bodyLower string, pos int) bool {
	windowStart := pos - 20
	if windowStart < 0 {
		windowStart = 0
	}
	window := bodyLower[windowStart:pos]

	for _, kw := range codeKeywords {
		if strings.Contains(window, kw) {
			return true
		}
	}
	return false
}

func ExtractLinks(body string) []string {
	var links []string

	urlMatches := urlRe.FindAllString(body, -1)
	links = append(links, urlMatches...)

	htmlLinks := extractHTMLLinks(body)
	links = append(links, htmlLinks...)

	return deduplicate(links)
}

func extractHTMLLinks(body string) []string {
	var links []string

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return links
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return links
}

func PrimaryLink(body string) string {
	links := ExtractLinks(body)
	if len(links) == 0 {
		return ""
	}

	for _, link := range links {
		linkLower := strings.ToLower(link)
		for _, kw := range primaryKeywords {
			if strings.Contains(linkLower, kw) {
				return link
			}
		}
	}

	doc, err := html.Parse(strings.NewReader(body))
	if err == nil {
		var checkAnchor func(*html.Node) string
		checkAnchor = func(n *html.Node) string {
			if n.Type == html.ElementNode && n.Data == "a" {
				var href, text string
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						href = attr.Val
					}
				}
				text = getText(n)
				textLower := strings.ToLower(text)
				for _, kw := range primaryKeywords {
					if strings.Contains(textLower, kw) && href != "" && strings.HasPrefix(href, "http") {
						return href
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if result := checkAnchor(c); result != "" {
					return result
				}
			}
			return ""
		}
		if result := checkAnchor(doc); result != "" {
			return result
		}
	}

	return links[0]
}

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += getText(c)
	}
	return text
}

func ExtractAll(body string) Result {
	codes := ExtractCodes(body)
	links := ExtractLinks(body)
	primary := PrimaryLink(body)

	return Result{
		Codes:       codes,
		Links:       links,
		PrimaryLink: primary,
	}
}

func deduplicate(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, v := range input {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
