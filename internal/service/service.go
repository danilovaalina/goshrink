package service

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/danilovaalina/goshrink/internal/model"
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/zerolog/log"
)

type Repository interface {
	CreateLink(ctx context.Context, link model.Link) (model.Link, error)
	Link(ctx context.Context, shortCode string) (model.Link, error)
	SaveClick(ctx context.Context, click model.Click) error
	Analytics(ctx context.Context, shortCode string) (model.Analytics, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ShortURL(ctx context.Context, link model.Link) (model.Link, error) {
	isCustom := link.ShortCode != ""

	if !isCustom {
		shortCode, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 6)
		if err != nil {
			return model.Link{}, err
		}
		link.ShortCode = shortCode
	}

	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}

	newLink, err := s.repo.CreateLink(ctx, link)
	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			if isCustom {
				return model.Link{}, model.ErrCustomCodeTaken
			}

			link.ShortCode = ""
			return s.ShortURL(ctx, link)
		}
		return model.Link{}, err
	}

	return newLink, nil
}

func (s *Service) OriginURL(ctx context.Context, shortCode string, click model.Click) (string, error) {
	link, err := s.repo.Link(ctx, shortCode)
	if err != nil {
		return "", err
	}

	go func() {
		click.ID = uuid.New()
		click.LinkID = link.ID
		if err = s.repo.SaveClick(context.Background(), click); err != nil {
			log.Error().Err(err).Msg("failed to save click analytics")
		}
	}()

	return link.OriginURL, nil
}

func (s *Service) Analytics(ctx context.Context, shortCode string) (model.Analytics, error) {
	return s.repo.Analytics(ctx, shortCode)
}
