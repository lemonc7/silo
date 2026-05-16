package service

import (
	"context"
	"fmt"
	"log"

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
			TmdbID:     item.TMDBID,
			Type:       string(item.Type),
			Title:      item.Title,
			AirDate:    item.AirDate,
			PosterPath: item.PosterPath,
		}); err != nil {
			return fmt.Errorf("upsert media %d: %w", item.TMDBID, err)
		}
	}

	log.Printf("[sync] media: %d items", len(items))
	return nil
}

func (s *MediaService) SyncTV(ctx context.Context) error {
	mediaList, err := s.repo.GetTVMedia(ctx)
	if err != nil {
		return fmt.Errorf("query tv media: %w", err)
	}

	for _, m := range mediaList {
		seasons, err := s.client.FetchSeasons(ctx, m.TmdbID)
		if err != nil {
			log.Printf("[sync] skip tv %d: fetch seasons: %v", m.TmdbID, err)
			continue
		}

		for _, season := range seasons {
			count, err := s.repo.CountEpisodes(ctx, repo.CountEpisodesParams{
				SeriesID:     m.ID,
				SeasonNumber: season.SeasonNumber,
			})
			if err != nil {
				return fmt.Errorf("count episodes: %w", err)
			}

			if count == season.EpisodeCount {
				continue
			}

			episodes, err := s.client.FetchEpisodes(ctx, m.TmdbID, season.SeasonNumber)
			if err != nil {
				log.Printf("[sync] skip tv %d s%02d: fetch episodes: %v", m.TmdbID, season.SeasonNumber, err)
				continue
			}

			if _, err := s.repo.UpsertSeason(ctx, repo.UpsertSeasonParams{
				SeriesID:     m.ID,
				SeasonNumber: season.SeasonNumber,
				EpisodeCount: season.EpisodeCount,
				AirDate:      season.AirDate,
				PosterPath:   season.PosterPath,
			}); err != nil {
				return fmt.Errorf("upsert season: %w", err)
			}

			seasonID, err := s.repo.GetSeasonID(ctx, repo.GetSeasonIDParams{
				SeriesID:     m.ID,
				SeasonNumber: season.SeasonNumber,
			})
			if err != nil {
				return fmt.Errorf("get season id: %w", err)
			}

			for _, ep := range episodes {
				if _, err := s.repo.UpsertEpisode(ctx, repo.UpsertEpisodeParams{
					SeasonID:      seasonID,
					EpisodeNumber: ep.EpisodeNumber,
					AirDate:       ep.AirDate,
				}); err != nil {
					return fmt.Errorf("upsert episode: %w", err)
				}
			}
			log.Printf("[sync] tv %d s%02d: %d episodes", m.TmdbID, season.SeasonNumber, len(episodes))
		}
	}

	return nil
}
