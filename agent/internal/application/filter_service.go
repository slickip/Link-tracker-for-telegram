package application

import (
	"strings"
	"unicode/utf8"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type FilterService struct {
	stopWords       []string
	excludedAuthors map[string]struct{}
	minLength       int
}

func NewFilterService(
	stopWords []string,
	excludedAuthors []string,
	minLength int,
) *FilterService {
	authors := make(map[string]struct{}, len(excludedAuthors))
	for _, author := range excludedAuthors {
		author = strings.ToLower(strings.TrimSpace(author))
		if author != "" {
			authors[author] = struct{}{}
		}
	}

	return &FilterService{
		stopWords:       normalizeKeywords(stopWords),
		excludedAuthors: authors,
		minLength:       minLength,
	}
}

func (s *FilterService) ShouldProcess(update api.LinkUpdate) bool {
	description := strings.TrimSpace(update.Description)

	if utf8.RuneCountInString(description) < s.minLength {
		return false
	}

	username := strings.ToLower(strings.TrimSpace(update.Username))
	if _, ok := s.excludedAuthors[username]; ok {
		return false
	}

	words := toWordSet(description)
	for _, stopWord := range s.stopWords {
		if _, ok := words[stopWord]; ok {
			return false
		}
	}

	return true
}
