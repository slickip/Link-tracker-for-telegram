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
	normalizedStopWords := make([]string, 0, len(stopWords))
	for _, word := range stopWords {
		word = strings.ToLower(strings.TrimSpace(word))
		if word != "" {
			normalizedStopWords = append(normalizedStopWords, word)
		}
	}

	authors := make(map[string]struct{}, len(excludedAuthors))
	for _, author := range excludedAuthors {
		author = strings.ToLower(strings.TrimSpace(author))
		if author != "" {
			authors[author] = struct{}{}
		}
	}

	return &FilterService{
		stopWords:       normalizedStopWords,
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

	lowerDescription := strings.ToLower(description)

	for _, stopWord := range s.stopWords {
		if strings.Contains(lowerDescription, stopWord) {
			return false
		}
	}

	return true
}
