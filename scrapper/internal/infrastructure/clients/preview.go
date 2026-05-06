package clients

import (
	"html"
	"regexp"
	"strings"
)

const previewLimit = 200

var htmlTagRegexp = regexp.MustCompile(`<[^>]*>`)

func MakePreview(text string) string {
	text = cleanupText(text)

	runes := []rune(text)
	if len(runes) <= previewLimit {
		return text
	}

	return string(runes[:previewLimit])
}

func cleanupText(text string) string {
	text = html.UnescapeString(text)
	text = htmlTagRegexp.ReplaceAllString(text, " ")
	text = strings.Join(strings.Fields(text), " ")

	return strings.TrimSpace(text)
}
