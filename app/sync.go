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

type Service struct {
	repo    repo.Querier
	catalog catalog.Provider
	release release.Provider
}

func NewService(repo repo.Querier, catalog catalog.Provider, release release.Provider) *Service {
	return &Service{repo: repo, catalog: catalog, release: release}
}

func (s *Service) SyncMedia(ctx context.Context) error {
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

func (s *Service) SyncSeason(ctx context.Context) error {
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

func (s *Service) SyncEpisode(ctx context.Context) error {
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

func (s *Service) SyncMoviePage(ctx context.Context) error {
	movies, err := s.repo.GetMoviesWithoutPage(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取需要同步资源详情页的电影: %w", err)
	}

	for _, m := range movies {
		result, err := s.release.Resolve(ctx, release.Media{
			Type:  catalog.MediaTypeMovie,
			Title: m.Title,
			Year:  m.AirDate.Year(),
		})
		if err != nil {
			log.Printf("[sync] 获取电影[%s]资源链接失败: %v", m.Title, err)
			continue
		}
		_, err = s.repo.UpsertPages(ctx, repo.UpsertPagesParams{
			Provider:   "bt",
			MediaID:    m.ID,
			SeasonID:   nil,
			DetailPath: result,
		})
		if err != nil {
			log.Printf("[sync] 插入电影[%s]资源链接失败: %v", m.Title, err)
		}
	}

	return nil
}

func (s *Service) SyncSeriesPage(ctx context.Context) error {
	seasons, err := s.repo.GetSeasonsWithoutPage(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取需要同步资源详情页的季: %w", err)
	}

	for _, season := range seasons {
		result, err := s.release.Resolve(ctx, release.Media{
			Title: season.Title,
			Type:  catalog.MediaType(season.Type),
			Year:  season.AirDate.Year(),
		})

		if err != nil {
			log.Printf("[sync] 获取剧集[%s]资源链接失败: %v", season.Title, err)
			continue
		}

		if _, err := s.repo.UpsertPages(ctx, repo.UpsertPagesParams{
			Provider:   "bt",
			MediaID:    season.SeriesID,
			SeasonID:   &season.SeasonID,
			DetailPath: result,
		}); err != nil {
			log.Printf("[sync] 插入剧集[%s-%d]资源链接失败: %v", season.Title, season.SeasonNumber, err)
		}
	}

	return nil
}

func (s *Service) SyncMovieMagnets(ctx context.Context) error {
	items, err := s.repo.GetMoviePages(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取电影详情页链接: %w", err)
	}
	for _, item := range items {
		ts, err := s.release.FetchReleases(ctx, release.Resource{
			Target:   item.DetailPath,
			Type:     catalog.MediaTypeMovie,
			SeasonID: nil,
		})
		if err != nil {
			log.Printf("[sync] 获取磁力链接失败: %v", err)
			continue
		}

		fmt.Println(ts)
	}

	return nil
}
