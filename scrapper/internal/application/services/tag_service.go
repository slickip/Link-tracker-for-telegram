package services

import (
	"context"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
)

type TagService struct {
	tagRepo  repositories.TagRepository
	chatRepo repositories.ChatRepository
}

func NewTagService(tagRepo repositories.TagRepository, chatRepo repositories.ChatRepository) *TagService {
	return &TagService{
		tagRepo:  tagRepo,
		chatRepo: chatRepo,
	}
}

func (s *TagService) CreateTag(ctx context.Context, chatID int64, name string) error {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return pkg.ErrChatNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return pkg.ErrInvalidRequest
	}

	return s.tagRepo.Create(ctx, chatID, name)
}

func (s *TagService) ListTags(ctx context.Context, chatID int64) ([]domain.Tag, error) {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, pkg.ErrChatNotFound
	}

	return s.tagRepo.List(ctx, chatID)
}

func (s *TagService) RenameTag(ctx context.Context, chatID int64, oldName, newName string) error {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return pkg.ErrChatNotFound
	}

	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)

	if oldName == "" || newName == "" {
		return pkg.ErrInvalidRequest
	}

	return s.tagRepo.Rename(ctx, chatID, oldName, newName)
}

func (s *TagService) DeleteTag(ctx context.Context, chatID int64, name string) error {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return pkg.ErrChatNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return pkg.ErrInvalidRequest
	}

	return s.tagRepo.Delete(ctx, chatID, name)
}
