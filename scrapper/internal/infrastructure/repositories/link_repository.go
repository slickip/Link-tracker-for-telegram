package repositories

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type LinkRepository interface {
	Add(chatID int64, link domain.Link) error

	Remove(chatID int64, url string) error

	List(chatID int64) ([]domain.Link, error)

	ListByTag(chatID int64, tag string) ([]domain.Link, error)

	FindSubscribers(url string) ([]int64, error)

	GetAllTrackedLinks() ([]domain.Link, error)

	UpdateLastUpdated(url string, t time.Time) error
}

type InMemoryLinkRepository struct {
	data map[int64]map[string]domain.Link
}

func NewInMemoryLinkRepository() *InMemoryLinkRepository {
	return &InMemoryLinkRepository{
		data: make(map[int64]map[string]domain.Link),
	}
}

func (r *InMemoryLinkRepository) Add(chatID int64, link domain.Link) error {
	if r.data[chatID] == nil {
		r.data[chatID] = make(map[string]domain.Link)
	}

	if _, exists := r.data[chatID][link.URL]; exists {
		return pkg.ErrLinkAlreadyTracked
	}

	r.data[chatID][link.URL] = link

	return nil
}

func (r *InMemoryLinkRepository) Remove(chatID int64, url string) error {

	if r.data[chatID] == nil {
		return pkg.ErrLinkNotTracked
	}

	delete(r.data[chatID], url)

	return nil
}

func (r *InMemoryLinkRepository) List(chatID int64) ([]domain.Link, error) {

	links := r.data[chatID]

	result := make([]domain.Link, 0, len(links))

	for _, link := range links {
		result = append(result, link)
	}

	return result, nil
}

func (r *InMemoryLinkRepository) ListByTag(chatID int64, tag string) ([]domain.Link, error) {

	var result []domain.Link

	for _, link := range r.data[chatID] {

		for _, t := range link.Tags {
			if t == tag {
				result = append(result, link)
			}
		}

	}

	return result, nil
}

func (r *InMemoryLinkRepository) FindSubscribers(url string) ([]int64, error) {

	var result []int64

	for chatID, links := range r.data {

		if _, ok := links[url]; ok {
			result = append(result, chatID)
		}

	}

	return result, nil
}

func (r *InMemoryLinkRepository) GetAllTrackedLinks() ([]domain.Link, error) {

	unique := make(map[string]domain.Link)

	for _, links := range r.data {

		for url, link := range links {
			unique[url] = link
		}

	}

	result := make([]domain.Link, 0, len(unique))

	for _, link := range unique {
		result = append(result, link)
	}

	return result, nil
}

func (r *InMemoryLinkRepository) UpdateLastUpdated(url string, t time.Time) error {

	for chatID, links := range r.data {

		if link, ok := links[url]; ok {

			link.LastUpdatedAt = t
			r.data[chatID][url] = link

		}

	}

	return nil
}
