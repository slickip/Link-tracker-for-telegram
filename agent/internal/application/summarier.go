package application

import "context"

type Summarizer interface {
	Summarize(ctx context.Context, text string, threshold int) (string, error)
}
