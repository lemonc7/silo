package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/download"
	"github.com/lemonc7/silo/release"
	"github.com/lemonc7/silo/repo"
)

type Service struct {
	db      *sql.DB
	repo    *repo.Queries
	catalog catalog.Provider
	release release.Provider
	down    download.Downloader
}

func NewService(db *sql.DB, catalog catalog.Provider, release release.Provider, down download.Downloader) *Service {
	return &Service{
		db:      db,
		repo:    repo.New(db),
		catalog: catalog,
		release: release,
		down:    down,
	}
}

func (s *Service) InitProfilePriority(ctx context.Context, profiles []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("启动事务: %w", err)
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	for i, p := range profiles {
		if _, err := qtx.UpsertProfilePriority(ctx, repo.UpsertProfilePriorityParams{Profile: p, Priority: int64(i)}); err != nil {
			return fmt.Errorf("插入标签优先级: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	slog.Info("标签优先级同步完成", "component", "sync")
	return nil
}

func (s *Service) SyncMedia(ctx context.Context) error {
	items, err := s.catalog.FetchMedia(ctx)
	if err != nil {
		return fmt.Errorf("获取待同步的媒体: %w", err)
	}

	for _, item := range items {
		if _, err := s.repo.UpsertMedia(ctx, repo.UpsertMediaParams{
			TmdbID:     item.TmdbID,
			Type:       string(item.Type),
			Title:      item.Title,
			AirDate:    item.AirDate,
			PosterPath: item.PosterPath,
		}); err != nil {
			return fmt.Errorf("插入媒体(%s)到数据库: %w", item.Title, err)
		}
	}

	slog.Info("媒体同步完成", "component", "sync")
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
			slog.Warn("获取季信息失败", "component", "catalog", "tmdb_id", t.TmdbID, "err", err)
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
				slog.Warn("写入季信息失败", "component", "db", "tmdb_id", t.TmdbID, "season_number", season.SeasonNumber, "err", err)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	slog.Info("剧集季信息同步完成", "component", "sync")
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
			slog.Warn("获取集信息失败", "component", "catalog", "season_id", se.ID, "tmdb_id", se.TmdbID, "err", err)
			continue
		}

		for _, ep := range episodes {
			if _, err := s.repo.UpsertEpisode(ctx, repo.UpsertEpisodeParams{
				SeasonID:      se.ID,
				EpisodeNumber: ep.EpisodeNumber,
				AirDate:       ep.AirDate,
			}); err != nil {
				slog.Warn("写入集信息失败", "component", "db", "season_id", se.ID, "episode_number", ep.EpisodeNumber, "err", err)
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	slog.Info("剧集集信息同步完成", "component", "sync")
	return nil
}

func (s *Service) SyncMoviePage(ctx context.Context) error {
	movies, err := s.repo.GetMoviesWithoutPage(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取电影信息(未同步资源详情页链接): %w", err)
	}

	for _, m := range movies {
		result, err := s.release.Resolve(ctx, release.Media{Type: catalog.MediaTypeMovie, Title: m.Title, Year: m.AirDate.Year()})
		if err != nil {
			slog.Warn("解析电影详情页失败", "component", "release", "title", m.Title, "year", m.AirDate.Year(), "err", err)
			continue
		}
		if _, err = s.repo.UpsertPages(ctx, repo.UpsertPagesParams{Provider: "bt", MediaID: m.ID, SeasonID: nil, DetailPath: result}); err != nil {
			slog.Warn("写入电影详情页失败", "component", "db", "title", m.Title, "detail_path", result, "err", err)
		}
	}

	slog.Info("同步电影详情页完成", "component", "sync")
	return nil
}

func (s *Service) SyncSeriesPage(ctx context.Context) error {
	seasons, err := s.repo.GetSeasonsWithoutPage(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取剧集信息(未同步资源详情页链接): %w", err)
	}

	for _, season := range seasons {
		result, err := s.release.Resolve(ctx, release.Media{Title: season.Title, Type: catalog.MediaType(season.Type), Year: season.AirDate.Year()})
		if err != nil {
			slog.Warn("解析剧集详情页失败", "component", "release", "title", season.Title, "season_number", season.SeasonNumber, "err", err)
			continue
		}

		if _, err := s.repo.UpsertPages(ctx, repo.UpsertPagesParams{Provider: "bt", MediaID: season.SeriesID, SeasonID: &season.SeasonID, DetailPath: result}); err != nil {
			slog.Warn("写入剧集详情页失败", "component", "db", "title", season.Title, "season_number", season.SeasonNumber, "detail_path", result, "err", err)
		}
	}

	slog.Info("同步剧集季详情页完成", "component", "sync")
	return nil
}

func (s *Service) SyncMovieMagnets(ctx context.Context) error {
	items, err := s.repo.GetMoviePages(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取电影详情页链接: %w", err)
	}
	for _, item := range items {
		ts, err := s.release.FetchReleases(ctx, item.DetailPath)
		if err != nil {
			slog.Warn("获取电影磁力链接失败", "component", "release", "media_id", item.MediaID, "detail_path", item.DetailPath, "err", err)
			continue
		}

		for _, t := range ts {
			if _, err := s.repo.UpsertMagnets(ctx, repo.UpsertMagnetsParams{MediaID: item.MediaID, Title: t.Title, MagnetUrl: t.Magnet, SizeMb: t.Size, Seeder: t.Seeder, Profile: t.Profile}); err != nil {
				slog.Warn("写入电影磁力链接失败", "component", "db", "media_id", item.MediaID, "magnet", t.Magnet, "err", err)
				continue
			}
		}
	}

	slog.Info("同步电影磁力链接完成", "component", "sync")
	return nil
}

func (s *Service) SyncSeriesMagnets(ctx context.Context) error {
	items, err := s.repo.GetSeasonPages(ctx, "bt")
	if err != nil {
		return fmt.Errorf("获取剧集季详情页链接: %w", err)
	}

	for _, item := range items {
		ts, err := s.release.FetchReleases(ctx, item.DetailPath)
		if err != nil {
			slog.Warn("获取剧集季磁力链接失败", "component", "release", "media_id", item.MediaID, "season_id", item.SeasonID, "detail_path", item.DetailPath, "err", err)
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("启动事务: %w", err)
		}
		qtx := s.repo.WithTx(tx)

		for _, t := range ts {
			id, err := qtx.UpsertMagnets(ctx, repo.UpsertMagnetsParams{MediaID: item.MediaID, SeasonID: &item.SeasonID, Title: t.Title, MagnetUrl: t.Magnet, SizeMb: t.Size, Seeder: t.Seeder, Profile: t.Profile})
			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("写入磁力链接: %w", err)
			}
			if len(t.Episodes) == 1 && t.Episodes[0] == math.MaxInt64 {
				if _, err := qtx.UpsertMagnetEpisodeBySeasonID(ctx, repo.UpsertMagnetEpisodeBySeasonIDParams{MagnetID: id, SeasonID: item.SeasonID}); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("绑定整季磁力链接到对应集: %w", err)
				}
			} else {
				for _, ep := range t.Episodes {
					if _, err := qtx.UpsertMagnetEpisodeByEpisodeNumber(ctx, repo.UpsertMagnetEpisodeByEpisodeNumberParams{MagnetID: id, SeasonID: item.SeasonID, EpisodeNumber: ep}); err != nil {
						_ = tx.Rollback()
						return fmt.Errorf("绑定磁力链接到对应集: %w", err)
					}
				}
			}
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			slog.Error("提交剧集磁力链接事务失败", "component", "sync", "media_id", item.MediaID, "season_id", item.SeasonID, "err", err)
		}
	}

	slog.Info("同步剧集磁力链接成功", "component", "sync")
	return nil
}
