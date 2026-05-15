package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ── Client 接口 ──────────────────────────────────

// Client TMDB 数据源接口。
type Client interface {
	FetchWatchlist(ctx context.Context, mediaType MediaType) ([]MediaItem, error)
	FetchEpisodes(ctx context.Context, tmdbID int, season int) ([]Episode, error)
	FetchTVShow(ctx context.Context, tmdbID int) (*TVShow, error)
}

// ── HTTPClient 实现 ─────────────────────────────

// HTTPClient 基于 TMDB v3 API + Bearer Token 认证。
// Bearer Token（API Read Access Token）在 https://www.themoviedb.org/settings/api 获取，长期有效。
type HTTPClient struct {
	bearerToken string
	accountID   string
	baseURL     string
	client      *http.Client
}

// NewHTTPClient 创建 TMDB API 客户端。
// bearerToken: API Read Access Token，永久有效。
// accountID: 如果为空，自动通过 /account 接口获取。
func NewHTTPClient(bearerToken, accountID string) (*HTTPClient, error) {
	c := &HTTPClient{
		bearerToken: bearerToken,
		accountID:   accountID,
		baseURL:     "https://api.themoviedb.org/3",
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	if c.accountID == "" {
		id, err := c.fetchAccountID(context.Background())
		if err != nil {
			return nil, fmt.Errorf("fetch account id: %w", err)
		}
		c.accountID = id
	}

	log.Printf("[tmdb] ready: account_id=%s", c.accountID)
	return c, nil
}

func (c *HTTPClient) fetchAccountID(ctx context.Context) (string, error) {
	resp, err := c.get(ctx, "/account", nil)
	if err != nil {
		return "", err
	}

	var body struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(resp, &body); err != nil {
		return "", err
	}
	return strconv.Itoa(body.ID), nil
}

// ── FetchWatchlist ───────────────────────────────

func (c *HTTPClient) FetchWatchlist(ctx context.Context, mediaType MediaType) ([]MediaItem, error) {
	endpoint := fmt.Sprintf("/account/%s/watchlist/%s", c.accountID, string(mediaType))

	var all []MediaItem
	page := 1

	for {
		params := url.Values{}
		params.Set("language", "zh-CN")
		params.Set("sort_by", "created_at.asc")
		params.Set("page", strconv.Itoa(page))

		resp, err := c.get(ctx, endpoint, params)
		if err != nil {
			return nil, fmt.Errorf("fetch watchlist page %d: %w", page, err)
		}

		var body apiWatchlistResponse
		if err := json.Unmarshal(resp, &body); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		for _, item := range body.Results {
			all = append(all, c.toMediaItem(item, mediaType))
		}

		log.Printf("[tmdb] watchlist/%s page %d/%d (%d items so far)",
			mediaType, page, body.TotalPages, len(all))

		if page >= body.TotalPages {
			break
		}
		page++
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("[tmdb] watchlist/%s done: %d items", mediaType, len(all))
	return all, nil
}

// ── FetchEpisodes ────────────────────────────────

func (c *HTTPClient) FetchEpisodes(ctx context.Context, tmdbID int, season int) ([]Episode, error) {
	endpoint := fmt.Sprintf("/tv/%d/season/%d", tmdbID, season)
	params := url.Values{}
	params.Set("language", "zh-CN")

	resp, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("fetch season %d for tv %d: %w", season, tmdbID, err)
	}

	var body apiSeasonResponse
	if err := json.Unmarshal(resp, &body); err != nil {
		return nil, fmt.Errorf("decode season response: %w", err)
	}

	episodes := make([]Episode, 0, len(body.Episodes))
	for _, ep := range body.Episodes {
		episodes = append(episodes, Episode{
			SeasonNum:  ep.SeasonNum,
			EpisodeNum: ep.EpisodeNum,
			AirDate:    ep.AirDate,
			Name:       ep.Name,
		})
	}

	return episodes, nil
}

// ── FetchTVShow ──────────────────────────────────

func (c *HTTPClient) FetchTVShow(ctx context.Context, tmdbID int) (*TVShow, error) {
	endpoint := fmt.Sprintf("/tv/%d", tmdbID)
	params := url.Values{}
	params.Set("language", "zh-CN")

	resp, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("fetch tv show %d: %w", tmdbID, err)
	}

	var body apiTVShowResponse
	if err := json.Unmarshal(resp, &body); err != nil {
		return nil, fmt.Errorf("decode tv show response: %w", err)
	}

	seasons := make([]Season, 0, len(body.Seasons))
	for _, s := range body.Seasons {
		if s.SeasonNum == 0 {
			continue
		}
		seasons = append(seasons, Season{
			SeasonNum:    s.SeasonNum,
			EpisodeCount: s.EpisodeCount,
			AirDate:      s.AirDate,
			Name:         s.Name,
		})
	}

	return &TVShow{
		TMDBID:    body.ID,
		Name:      body.Name,
		Seasons:   seasons,
		PosterURL: buildImageURL(body.PosterPath),
		Overview:  body.Overview,
		Year:      extractYear(body.FirstAirDate),
	}, nil
}

// ── 内部方法 ─────────────────────────────────────

func (c *HTTPClient) get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			StatusMessage string `json:"status_message"`
			StatusCode    int    `json:"status_code"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("tmdb api error %d: %s", resp.StatusCode, errResp.StatusMessage)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *HTTPClient) toMediaItem(item apiWatchlistItem, mediaType MediaType) MediaItem {
	mi := MediaItem{
		TMDBID:   item.ID,
		Type:     mediaType,
		Overview: item.Overview,
	}

	if mediaType == MediaTypeMovie {
		mi.Title = item.Title
		mi.OriginalTitle = item.OriginalTitle
		mi.Year = extractYear(item.ReleaseDate)
		mi.PosterURL = buildImageURL(item.PosterPath)
	} else {
		mi.Title = item.Name
		mi.OriginalTitle = item.OriginalName
		mi.Year = extractYear(item.FirstAirDate)
		mi.PosterURL = buildImageURL(item.PosterPath)
	}

	return mi
}

// ── 工具函数 ─────────────────────────────────────

func buildImageURL(path string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/w500" + strings.TrimPrefix(path, "/")
}

func extractYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	y, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return y
}
