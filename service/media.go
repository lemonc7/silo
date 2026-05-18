package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemonc7/silo/media"
	"github.com/lemonc7/silo/repo"
)

type MediaService struct {
	repo   repo.Querier
	client media.Client
}

func NewMediaService(r repo.Querier, c media.Client) *MediaService {
	return &MediaService{repo: r, client: c}
}

func (s *MediaService) SyncMedia(ctx context.Context) error {
	items, err := s.client.FetchMedia(ctx)
	if err != nil {
		return fmt.Errorf("fetch media: %w", err)
	}

	for _, item := range items {
		if _, err := s.repo.UpsertMedia(ctx, repo.UpsertMediaParams{
			TmdbID:     item.TmdbID,
			Type:       string(item.Type),
			Title:      item.Title,
			AirDate:    item.AirDate,
			PosterPath: item.PosterPath,
		}); err != nil {
			return fmt.Errorf("upsert media %d: %w", item.TmdbID, err)
		}
	}

	log.Printf("[sync] media: %d items", len(items))
	return nil
}

func (s *MediaService) SyncSeason(ctx context.Context) error {
	tvs, err := s.repo.GetOutOfSyncTVs(ctx)
	if err != nil {
		return fmt.Errorf("get out of sync tv: %w", err)
	}

	for _, t := range tvs {
		seasons, err := s.client.FetchSeasons(ctx, t.TmdbID)
		if err != nil {
			log.Printf("[sync] skip tv %d: fetch seasons: %v", t.TmdbID, err)
			continue
		}

		for _, season := range seasons {
			_, err := s.repo.UpsertSeason(ctx, repo.UpsertSeasonParams{
				SeriesID:     t.ID,
				SeasonNumber: season.SeasonNumber,
				EpisodeCount: season.EpisodeCount,
				AirDate:      season.AirDate,
				PosterPath:   season.PosterPath,
			})
			if err != nil {
				log.Printf("[sync] skip tv %d: upsert season(%d): %v", t.TmdbID, season.SeasonNumber, err)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

func (s *MediaService) SyncEpisode(ctx context.Context) error {
	seasons, err := s.repo.GetOutOfSyncSeasons(ctx)
	if err != nil {
		return fmt.Errorf("get out of sync seasons: %w", err)
	}

	for _, se := range seasons {
		episodes, err := s.client.FetchEpisodes(ctx, se.TmdbID, se.SeasonNumber)
		if err != nil {
			log.Printf("[sync] skip season %d: fetch episodes: %v", se.ID, err)
			continue
		}

		for _, ep := range episodes {
			if _, err := s.repo.UpsertEpisode(ctx, repo.UpsertEpisodeParams{
				SeasonID:      se.ID,
				EpisodeNumber: ep.EpisodeNumber,
				AirDate:       ep.AirDate,
			}); err != nil {
				log.Printf("[sync] skip episode %d: upsert: %v", ep.EpisodeNumber, err)
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
	return nil
}
