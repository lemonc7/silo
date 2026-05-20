package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/release"
	"github.com/lemonc7/silo/repo"
)

type SyncService struct {
	repo    repo.Querier
	catalog catalog.Provider
	release release.Provider
}

func NewMediaService(repo repo.Querier, catalog catalog.Provider, release release.Provider) *SyncService {
	return &SyncService{repo: repo, catalog: catalog, release: release}
}

func (s *SyncService) SyncMedia(ctx context.Context) error {
	items, err := s.catalog.FetchMedia(ctx)
	if err != nil {
		return fmt.Errorf("获取TMDB媒体信息: %w", err)
	}

	for _, item := range items {
		if _, err := s.repo.UpsertMedia(ctx, repo.UpsertMediaParams{
			TmdbID:     item.TmdbID,
			Type:       string(item.Type),
			Title:      item.Title,
			AirDate:    item.AirDate,
			PosterPath: item.PosterPath,
		}); err != nil {
			return fmt.Errorf("插入TMDB媒体[%d]到数据库: %w", item.TmdbID, err)
		}
	}

	log.Printf("[sync] 媒体已同步 %d 条", len(items))
	return nil
}

func (s *SyncService) SyncSeason(ctx context.Context) error {
	tvs, err := s.repo.GetOutOfSyncTVs(ctx)
	if err != nil {
		return fmt.Errorf("获取待同步的剧集: %w", err)
	}

	for _, t := range tvs {
		seasons, err := s.catalog.FetchSeasons(ctx, t.TmdbID)
		if err != nil {
			log.Printf("[sync] 跳过剧集 %d: 获取季信息失败: %v", t.TmdbID, err)
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
				log.Printf("[sync] 跳过剧集 %d: 插入季(%d)信息失败: %v", t.TmdbID, season.SeasonNumber, err)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

func (s *SyncService) SyncEpisode(ctx context.Context) error {
	seasons, err := s.repo.GetOutOfSyncSeasons(ctx)
	if err != nil {
		return fmt.Errorf("获取待同步的季: %w", err)
	}

	for _, se := range seasons {
		episodes, err := s.catalog.FetchEpisodes(ctx, se.TmdbID, se.SeasonNumber)
		if err != nil {
			log.Printf("[sync] 跳过季 %d: 获取集信息失败: %v", se.ID, err)
			continue
		}

		for _, ep := range episodes {
			if _, err := s.repo.UpsertEpisode(ctx, repo.UpsertEpisodeParams{
				SeasonID:      se.ID,
				EpisodeNumber: ep.EpisodeNumber,
				AirDate:       ep.AirDate,
			}); err != nil {
				log.Printf("[sync] 跳过集 %d: 插入数据库失败: %v", ep.EpisodeNumber, err)
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func (s *SyncService) SyncResourceLink(ctx context.Context) error {
	items, err := s.repo.GetOutOfMedias(ctx)
	if err != nil {
		return fmt.Errorf("获取需要爬取资源的媒体: %w", err)
	}

	for _, item := range items {
		_, err := s.release.Resolve(ctx, release.Media{
			Type:  catalog.MediaType(item.Type),
			Title: item.Title,
			Year:  item.AirDate.Year(),
		})
		if err != nil {
			fmt.Printf("[bt] 获取资源详情页链接失败: %v", err)
			continue
		}

	}

	return nil
}
