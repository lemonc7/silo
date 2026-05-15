package service

import (
	"context"
	"log"
	"time"

	"github.com/lemonc7/silo/download"
	"github.com/lemonc7/silo/search"
	"github.com/lemonc7/silo/store"
	"github.com/lemonc7/silo/tmdb"
)

// SyncService 核心同步编排层。
// 组合 TMDB → Matcher → Scraper → Downloader → Store 全链路。
type SyncService struct {
	tmdb       tmdb.Client
	pool       *search.Pool
	scraper    search.Scraper
	matcher    *search.Matcher
	downloader download.Downloader
	store      store.Store
}

// NewSyncService 创建同步服务。
func NewSyncService(
	tmdbClient tmdb.Client,
	pool *search.Pool,
	scraper search.Scraper,
	downloader download.Downloader,
	store store.Store,
) *SyncService {
	return &SyncService{
		tmdb:       tmdbClient,
		pool:       pool,
		scraper:    scraper,
		matcher:    search.NewMatcher(),
		downloader: downloader,
		store:      store,
	}
}

// SyncAll 核心入口：拉取 TMDB 收藏 → 对比本地 DB → 逐个搜索 → 匹配 → 下载。
func (s *SyncService) SyncAll(ctx context.Context) error {
	log.Println("[sync] ======== sync start ========")

	// ── 1. 拉取收藏列表 ────────────────────────
	movies, err := s.tmdb.FetchWatchlist(ctx, tmdb.MediaTypeMovie)
	if err != nil {
		log.Printf("[sync] fetch movie watchlist: %v", err)
	}
	tvs, err := s.tmdb.FetchWatchlist(ctx, tmdb.MediaTypeTV)
	if err != nil {
		log.Printf("[sync] fetch tv watchlist: %v", err)
	}

	var items []tmdb.MediaItem

	// 电影：直接加入待处理队列
	for _, m := range movies {
		downloaded, _ := s.store.IsDownloaded(ctx, m.TMDBID, 0, 0)
		if !downloaded {
			items = append(items, m)
		}
	}
	log.Printf("[sync] movies: %d in watchlist, %d to download", len(movies), len(items))

	// 剧集：逐部查季→查集→筛未下载
	tvDownloadCount := 0
	for _, tv := range tvs {
		episodes := s.expandTVEpisodes(ctx, tv)
		items = append(items, episodes...)
		tvDownloadCount += len(episodes)
	}
	log.Printf("[sync] tv: %d shows in watchlist, %d episodes to download", len(tvs), tvDownloadCount)

	log.Printf("[sync] total: %d items to process", len(items))

	// ── 2. 逐个处理 ────────────────────────────
	for i, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		s.processItem(ctx, item)
		// 反爬间隔
		if i < len(items)-1 {
			time.Sleep(30 * time.Second)
		}
	}

	log.Println("[sync] ======== sync done ========")
	return nil
}

// expandTVEpisodes 展开一部剧的所有待下载剧集。
// 流程：获取剧集详情 → 遍历所有季 → 获取每集的 AirDate → 筛选已播出+未下载。
func (s *SyncService) expandTVEpisodes(ctx context.Context, tv tmdb.MediaItem) []tmdb.MediaItem {
	show, err := s.tmdb.FetchTVShow(ctx, tv.TMDBID)
	if err != nil {
		log.Printf("[sync] fetch tv show %s (id=%d): %v", tv.Title, tv.TMDBID, err)
		return nil
	}

	var episodes []tmdb.MediaItem
	today := time.Now().Format("2006-01-02")

	for _, season := range show.Seasons {
		eps, err := s.tmdb.FetchEpisodes(ctx, tv.TMDBID, season.SeasonNum)
		if err != nil {
			log.Printf("[sync] fetch s%02d for %s: %v", season.SeasonNum, show.Name, err)
			continue
		}

		for _, ep := range eps {
			// 跳过未播出的（AirDate > 今天 或 无 AirDate）
			if ep.AirDate == "" || ep.AirDate > today {
				continue
			}

			// 跳过已下载的
			downloaded, _ := s.store.IsDownloaded(ctx, tv.TMDBID, ep.SeasonNum, ep.EpisodeNum)
			if downloaded {
				continue
			}

			episodes = append(episodes, tmdb.MediaItem{
				TMDBID:        tv.TMDBID,
				Title:         show.Name,
				OriginalTitle: show.Name,
				Type:          tmdb.MediaTypeTV,
				Year:          show.Year,
				Season:        ep.SeasonNum,
				Episode:       ep.EpisodeNum,
				PosterURL:     show.PosterURL,
				Overview:      show.Overview,
			})
		}
	}

	return episodes
}

// processItem 处理单个条目：搜索 → 匹配 → 取磁力 → 下载 → 记录。
func (s *SyncService) processItem(ctx context.Context, item tmdb.MediaItem) {
	// 幂等检查
	if downloaded, _ := s.store.IsDownloaded(ctx, item.TMDBID, item.Season, item.Episode); downloaded {
		log.Printf("[sync] skip %s S%02dE%02d (already downloaded)", item.Title, item.Season, item.Episode)
		return
	}

	// 获取 Page
	page, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Printf("[sync] acquire page: %v", err)
		return
	}
	defer s.pool.Release(page)

	// 搜索
	keyword := s.matcher.GenerateSearchKeyword(item)
	results, err := s.scraper.Search(ctx, page, keyword)
	if err != nil || len(results) == 0 {
		log.Printf("[sync] not found: %s", keyword)
		return
	}

	// 匹配
	best := s.matcher.BestMatch(item, results)
	if best == nil {
		log.Printf("[sync] no match among %d results: %s", len(results), keyword)
		return
	}
	log.Printf("[sync] matched: %s → %s", keyword, best.Title)

	// 提取磁力
	magnets, err := s.scraper.FetchMagnets(ctx, page, best.DetailURL)
	if err != nil || len(magnets) == 0 {
		log.Printf("[sync] no magnet: %s", best.Title)
		return
	}

	// 选最优磁力（做种数最多）
	bestMagnet := magnets[0]
	for _, m := range magnets[1:] {
		if m.Seeders > bestMagnet.Seeders {
			bestMagnet = m
		}
	}

	// 下载
	taskID, err := s.downloader.AddMagnet(ctx, bestMagnet.Magnet, "")
	if err != nil {
		log.Printf("[sync] download error: %v", err)
		return
	}

	// 记录
	_ = s.store.RecordDownload(ctx, store.DownloadRecord{
		TMDBID:       item.TMDBID,
		Type:         string(item.Type),
		Title:        item.Title,
		Season:       item.Season,
		Episode:      item.Episode,
		MagnetLink:   bestMagnet.Magnet,
		Status:       "downloading",
		DownloadedAt: time.Now(),
	})

	log.Printf("[sync] ok: %s S%02dE%02d → task=%s", item.Title, item.Season, item.Episode, taskID)
}
