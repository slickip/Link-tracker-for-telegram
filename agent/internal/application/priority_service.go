package application

import "strings"

type Priority string

const (
	PriorityHigh   Priority = "HIGH"
	PriorityMedium Priority = "MEDIUM"
	PriorityLow    Priority = "LOW"
)

type PriorityService struct {
	highKeywords []string
	lowKeywords  []string
}

func NewPriorityService(highKeywords []string, lowKeywords []string) *PriorityService {
	return &PriorityService{
		highKeywords: normalizeKeywords(highKeywords),
		lowKeywords:  normalizeKeywords(lowKeywords),
	}
}

func (s *PriorityService) DeterminePriority(text string) Priority {
	words := toWordSet(text)

	for _, keyword := range s.highKeywords {
		if _, ok := words[keyword]; ok {
			return PriorityHigh
		}
	}

	for _, keyword := range s.lowKeywords {
		if _, ok := words[keyword]; ok {
			return PriorityLow
		}
	}

	return PriorityMedium
}

func normalizeKeywords(keywords []string) []string {
	result := make([]string, 0, len(keywords))

	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" {
			result = append(result, keyword)
		}
	}

	return result
}

func toWordSet(text string) map[string]struct{} {
	fields := strings.FieldsFunc(strings.ToLower(text), isWordSeparator)

	words := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		words[field] = struct{}{}
	}

	return words
}

func isWordSeparator(r rune) bool {
	return !isWordRune(r)
}

func isWordRune(r rune) bool {
	return r >= 'a' && r <= 'z' ||
		r >= 'а' && r <= 'я' ||
		r == 'ё' ||
		r >= '0' && r <= '9'
}
