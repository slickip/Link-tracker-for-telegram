package ai

import "unicode/utf8"

func cutWithEllipsis(text string, threshold int) string {
	if utf8.RuneCountInString(text) <= threshold {
		return text
	}

	runes := []rune(text)
	return string(runes[:threshold]) + "..."
}
