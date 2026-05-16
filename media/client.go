package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/lemonc7/silo/config"
)

// Client TMDB 数据源接口。
type Client interface {
	FetchMedia(ctx context.Context) ([]MediaItem, error)
	FetchSeasons(ctx context.Context, tmdbID int64) ([]Season, error)
	FetchEpisodes(ctx context.Context, tmdbID, seasonNum int64) ([]Episode, error)
}

// HTTPClient 基于 TMDB v3 API + Bearer Token 认证。
type HTTPClient struct {
	bearerToken string
	accountID   string
	baseURL     string
	hosts       map[string]string
	proxyURL    string
	client      *http.Client
}

var _ Client = (*HTTPClient)(nil)

func NewHTTPClient(cfg config.TMDBConfig) (*HTTPClient, error) {
	c := &HTTPClient{
		bearerToken: cfg.BearerToken,
		accountID:   cfg.AccountID,
		baseURL:     "https://api.themoviedb.org/3",
		hosts:       cfg.Hosts,
		proxyURL:    cfg.Proxy,
	}
	c.buildClient()

	log.Printf("[tmdb] ready: account_id=%s", c.accountID)
	return c, nil
}

func (c *HTTPClient) buildClient() {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if ip, ok := c.hosts[host]; ok {
				addr = net.JoinHostPort(ip, port)
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	if c.proxyURL != "" {
		if u, err := url.Parse(c.proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	c.client = &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}
}

func (c *HTTPClient) FetchMedia(ctx context.Context) ([]MediaItem, error) {
	movie, err := c.FetchMovie(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch movie: %w", err)
	}

	tv, err := c.FetchTV(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch tv: %w", err)
	}

	return append(movie, tv...), nil
}

func (c *HTTPClient) FetchMovie(ctx context.Context) ([]MediaItem, error) {
	return c.fetchWatchlist(ctx, "movies")
}

func (c *HTTPClient) FetchTV(ctx context.Context) ([]MediaItem, error) {
	return c.fetchWatchlist(ctx, "tv")
}

func (c *HTTPClient) fetchWatchlist(ctx context.Context, media string) ([]MediaItem, error) {
	endpoint := fmt.Sprintf("/account/%s/watchlist/%s", c.accountID, media)

	params := url.Values{}
	params.Set("language", "zh-CN")
	params.Set("sort_by", "created_at.asc")

	var all []MediaItem
	for page := 1; ; page++ {
		params.Set("page", strconv.Itoa(page))

		var body mediaResponse
		if err := c.get(ctx, endpoint, params, &body); err != nil {
			return nil, fmt.Errorf("fetch watchlist/%s page %d: %w", media, page, err)
		}

		all = append(all, body.Results...)

		log.Printf("[tmdb] watchlist/%s page %d/%d (%d items)",
			media, page, body.TotalPages, len(all))

		if page >= body.TotalPages {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("[tmdb] watchlist/%s done: %d items", media, len(all))
	return all, nil
}

// FetchSeasons 拉取 TV 详情中的季信息。
func (c *HTTPClient) FetchSeasons(ctx context.Context, tmdbID int64) ([]Season, error) {
	endpoint := fmt.Sprintf("/tv/%d", tmdbID)
	params := url.Values{}
	params.Set("language", "zh-CN")

	var body seasonResponse
	if err := c.get(ctx, endpoint, params, &body); err != nil {
		return nil, fmt.Errorf("fetch tv detail %d: %w", tmdbID, err)
	}

	seasons := make([]Season, 0, len(body.Seasons))
	for _, s := range body.Seasons {
		if s.SeasonNumber == 0 {
			continue
		}
		seasons = append(seasons, Season{
			SeasonNumber: s.SeasonNumber,
			EpisodeCount: s.EpisodeCount,
			AirDate:      s.AirDate,
			PosterPath:   s.PosterPath,
		})
	}

	return seasons, nil
}

// FetchEpisodes 拉取指定季的剧集列表。
func (c *HTTPClient) FetchEpisodes(ctx context.Context, tmdbID, seasonNum int64) ([]Episode, error) {
	endpoint := fmt.Sprintf("/tv/%d/season/%d", tmdbID, seasonNum)
	params := url.Values{}
	params.Set("language", "zh-CN")

	var body episodesResponse
	if err := c.get(ctx, endpoint, params, &body); err != nil {
		return nil, fmt.Errorf("fetch s%02d episodes for tv %d: %w", seasonNum, tmdbID, err)
	}

	episodes := make([]Episode, 0, len(body.Episodes))
	for _, ep := range body.Episodes {
		episodes = append(episodes, Episode{
			EpisodeNumber: ep.EpisodeNumber,
			AirDate:       ep.AirDate,
		})
	}

	return episodes, nil
}

func (c *HTTPClient) get(ctx context.Context, endpoint string, params url.Values, target any) error {
	reqURL := c.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create tmdb request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send tmdb request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			StatusMessage string `json:"status_message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("tmdb api %d: %s", resp.StatusCode, errResp.StatusMessage)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
