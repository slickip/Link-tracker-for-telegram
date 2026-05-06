package parsers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	SourceGitHub        = "github"
	SourceStackOverflow = "stackoverflow"
)

type ParsedLink struct {
	RawURL string

	Source string

	GithubOwner string
	GithubRepo  string

	StackOverflowQuestionID int64
}

func ParseLink(raw string) (ParsedLink, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return ParsedLink{}, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return ParsedLink{}, fmt.Errorf("unsupported link scheme")
	}

	host := strings.ToLower(u.Hostname())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")

	if isHost(host, "github.com") && len(parts) >= 2 {
		return ParsedLink{
			RawURL:      raw,
			Source:      SourceGitHub,
			GithubOwner: parts[0],
			GithubRepo:  parts[1],
		}, nil
	}

	if isHost(host, "stackoverflow.com") &&
		len(parts) >= 2 &&
		parts[0] == "questions" {

		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return ParsedLink{}, err
		}

		return ParsedLink{
			RawURL:                  raw,
			Source:                  SourceStackOverflow,
			StackOverflowQuestionID: id,
		}, nil
	}

	return ParsedLink{}, fmt.Errorf("unsupported link")
}

func isHost(actual string, expected string) bool {
	return actual == expected || strings.HasSuffix(actual, "."+expected)
}
