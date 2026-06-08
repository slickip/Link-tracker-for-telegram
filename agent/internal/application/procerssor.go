package application

import (
	"context"
	"unicode/utf8"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

const DefaultPriority = "HIGH"

type Processor struct {
	filterService *FilterService
	summarizer    Summarizer
	threshold     int
}

func NewProcessor(
	filterService *FilterService,
	summarizer Summarizer,
	threshold int,
) *Processor {
	return &Processor{
		filterService: filterService,
		summarizer:    summarizer,
		threshold:     threshold,
	}
}

func (p *Processor) Process(
	ctx context.Context,
	update api.LinkUpdate,
) (api.LinkUpdate, bool, error) {
	if !p.filterService.ShouldProcess(update) {
		return api.LinkUpdate{}, false, nil
	}

	if utf8.RuneCountInString(update.Description) > p.threshold {
		summary, err := p.summarizer.Summarize(ctx, update.Description, p.threshold)
		if err != nil {
			return api.LinkUpdate{}, false, err
		}

		update.Description = summary
	}

	update.Priority = DefaultPriority

	return update, true, nil
}
