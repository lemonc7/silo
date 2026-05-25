package service

import (
	"context"

	"github.com/lemonc7/silo/repo"
)

func (s *Service) GetMedias(ctx context.Context) ([]repo.Media, error) {
	return s.repo.GetMedias(ctx)
}

func (s *Service) GetSeasons(ctx context.Context, seriesID int64) ([]repo.Season, error) {
	return s.repo.GetSeasons(ctx, seriesID)
}

func (s *Service) GetEpisodes(ctx context.Context, seasonID int64) ([]repo.Episode, error) {
	return s.repo.GetEpisodes(ctx, seasonID)
}
