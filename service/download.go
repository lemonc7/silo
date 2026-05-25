package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"github.com/lemonc7/silo/repo"
)

func (s *Service) CreateDownloadTasks(ctx context.Context) error {
	movies, err := s.repo.GetPendingMovies(ctx)
	if err != nil {
		return fmt.Errorf("获取待下载电影: %w", err)
	}
	for _, mediaID := range movies {
		magnet, err := s.repo.GetBestMagnetOfMovie(ctx, repo.GetBestMagnetOfMovieParams{
			MediaID:   mediaID,
			MinSizeMb: 0,
			MaxSizeMb: math.MaxFloat64,
		})
		if err != nil {
			if err != sql.ErrNoRows {
				slog.Warn("查询电影最佳磁力链接失败", "component", "db", "media_id", mediaID, "err", err)
			}
			continue
		}
		if err := s.createDownloadTask(ctx, magnet.ID, true); err != nil {
			return err
		}
	}

	seasons, err := s.repo.GetSeasonDownloadStats(ctx)
	if err != nil {
		return fmt.Errorf("获取待下载季统计: %w", err)
	}
	for _, season := range seasons {
		seasonID := season.SeasonID
		candidates, err := s.repo.GetSeasonMagnetCandidates(ctx, &seasonID)
		if err != nil {
			slog.Warn("查询季候选磁力链接失败", "component", "db", "season_id", season.SeasonID, "err", err)
			continue
		}
		if len(candidates) == 0 {
			continue
		}

		sortSeasonMagnetCandidates(candidates, season.MissingCount)
		if err := s.createDownloadTask(ctx, candidates[0].ID, false); err != nil {
			return err
		}
	}

	slog.Info("下载任务创建完成", "component", "sync")
	return nil
}

func sortSeasonMagnetCandidates(items []repo.GetSeasonMagnetCandidatesRow, missingCount int64) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]

		aMissingLeft := missingCount - a.MissingHitCount
		bMissingLeft := missingCount - b.MissingHitCount
		if aMissingLeft != bMissingLeft {
			return aMissingLeft < bMissingLeft
		}
		if a.ExtraCount != b.ExtraCount {
			return a.ExtraCount < b.ExtraCount
		}
		if a.MissingHitCount != b.MissingHitCount {
			return a.MissingHitCount > b.MissingHitCount
		}
		if a.TotalCoverCount != b.TotalCoverCount {
			return a.TotalCoverCount < b.TotalCoverCount
		}
		if a.ProfilePriority != b.ProfilePriority {
			return a.ProfilePriority < b.ProfilePriority
		}
		return a.Seeder > b.Seeder
	})
}

func (s *Service) createDownloadTask(ctx context.Context, magnetID int64, movie bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("启动事务: %w", err)
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	rows, err := qtx.CreateDownload(ctx, magnetID)
	if err != nil {
		return fmt.Errorf("创建下载任务: %w", err)
	}
	if rows == 0 {
		return nil
	}

	if movie {
		if _, err := qtx.MarkMovieDownloadingByMagnet(ctx, magnetID); err != nil {
			return fmt.Errorf("更新电影下载状态: %w", err)
		}
	} else {
		if _, err := qtx.MarkEpisodesDownloadingByMagnet(ctx, magnetID); err != nil {
			return fmt.Errorf("更新剧集下载状态: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}

func (s *Service) SubmitQueuedDownloads(ctx context.Context) error {
	items, err := s.repo.GetQueuedDownloads(ctx)
	if err != nil {
		return fmt.Errorf("获取待提交下载任务: %w", err)
	}

	for _, item := range items {
		if err := s.down.AddMagnet(ctx, item.MagnetUrl, ""); err != nil {
			if _, markErr := s.repo.MarkDownloadFailed(ctx, repo.MarkDownloadFailedParams{
				ID:    item.ID,
				Error: new(err.Error()),
			}); markErr != nil {
				slog.Warn("标记下载任务失败状态失败", "component", "db", "download_id", item.ID, "err", markErr)
			}
			slog.Warn("提交磁力链接到 qB 失败", "component", "qb", "download_id", item.ID, "magnet_id", item.MagnetID, "err", err)
			continue
		}

		if _, err := s.repo.MarkDownloadStarted(ctx, repo.MarkDownloadStartedParams{
			ID:     item.ID,
			QbHash: nil,
		}); err != nil {
			return fmt.Errorf("标记下载任务已开始: %w", err)
		}
	}

	slog.Info("queued 下载任务提交完成", "component", "sync", "count", len(items))
	return nil
}

func (s *Service) MarkDownloadCompleted(ctx context.Context, downloadID, magnetID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("启动事务: %w", err)
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	if _, err := qtx.MarkDownloadCompleted(ctx, downloadID); err != nil {
		return fmt.Errorf("标记下载任务完成: %w", err)
	}
	if _, err := qtx.MarkMovieCompletedByMagnet(ctx, magnetID); err != nil {
		return fmt.Errorf("更新电影完成状态: %w", err)
	}
	if _, err := qtx.MarkEpisodesDownloadedByMagnet(ctx, magnetID); err != nil {
		return fmt.Errorf("更新剧集完成状态: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}
