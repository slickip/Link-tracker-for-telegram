package parsers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
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
		return ParsedLink{}, fmt.Errorf("unsupported link")
	}

	host := u.Host
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")

	if strings.Contains(host, "github.com") && len(parts) >= 2 {

		return ParsedLink{
			RawURL:      raw,
			Source:      "github",
			GithubOwner: parts[0],
			GithubRepo:  parts[1],
		}, nil
	}

	if strings.Contains(host, "stackoverflow.com") &&
		len(parts) >= 2 &&
		parts[0] == "questions" {

		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return ParsedLink{}, err
		}

		return ParsedLink{
			RawURL:                  raw,
			Source:                  "stackoverflow",
			StackOverflowQuestionID: id,
		}, nil
	}

	return ParsedLink{}, fmt.Errorf("unsupported link")
}
