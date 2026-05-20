package catalog

import (
	"context"
	"encoding/json"
	"slices"
	"time"
)

type Provider interface {
	FetchMedia(ctx context.Context) ([]MediaItem, error)
	FetchSeasons(ctx context.Context, tmdbID int64) ([]Season, error)
	FetchEpisodes(ctx context.Context, tmdbID, seasonNum int64) ([]Episode, error)
}

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
)

type MediaItem struct {
	TmdbID     int64     `json:"tmdb_id"`
	Title      string    `json:"title"`
	Type       MediaType `json:"type"`
	AirDate    time.Time `json:"air_date"`
	PosterPath string    `json:"poster_path"`
}

func (m *MediaItem) UnmarshalJSON(data []byte) error {
	var r rawItem
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.TmdbID = r.ID
	m.PosterPath = r.PosterPath

	if r.ReleaseDate != "" {
		m.AirDate = parseDate(r.ReleaseDate)
	} else {
		m.AirDate = parseDate(r.FirstAirDate)
	}

	// 电影的标题是 title
	if r.Title != "" {
		m.Title = r.Title
		m.Type = MediaTypeMovie
		return nil
	}

	// 剧集的标题是 name
	m.Title = r.Name
	// 动漫 tag 是16
	if slices.Contains(r.GenreIDs, 16) {
		m.Type = MediaTypeAnime
		return nil
	}

	m.Type = MediaTypeTV
	return nil
}

type Season struct {
	SeasonNumber int64     `json:"season_number"`
	EpisodeCount int64     `json:"episode_count"`
	AirDate      time.Time `json:"air_date"`
	PosterPath   string    `json:"poster_path"`
}

func (s *Season) UnmarshalJSON(data []byte) error {
	var r seasonRaw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	s.SeasonNumber = r.SeasonNumber
	s.EpisodeCount = r.EpisodeCount
	s.AirDate = parseDate(r.AirDate)
	s.PosterPath = r.PosterPath
	return nil
}

type Episode struct {
	EpisodeNumber int64     `json:"episode_number"`
	AirDate       time.Time `json:"air_date"`
}

func (s *Episode) UnmarshalJSON(data []byte) error {
	var r episodeRaw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	s.EpisodeNumber = r.EpisodeNumber
	s.AirDate = parseDate(r.AirDate)
	return nil
}

type mediaResponse struct {
	Page         int         `json:"page"`
	Results      []MediaItem `json:"results"`
	TotalPages   int         `json:"total_pages"`
	TotalResults int         `json:"total_results"`
}

type seasonResponse struct {
	Status  string   `json:"status"`
	Seasons []Season `json:"seasons"`
}

type episodesResponse struct {
	Episodes []Episode `json:"episodes"`
}

type rawItem struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Name         string `json:"name"`
	ReleaseDate  string `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
	PosterPath   string `json:"poster_path"`
	GenreIDs     []int  `json:"genre_ids"`
	Status       string `json:"status"`
}

type seasonRaw struct {
	SeasonNumber int64  `json:"season_number"`
	EpisodeCount int64  `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type episodeRaw struct {
	EpisodeNumber int64  `json:"episode_number"`
	AirDate       string `json:"air_date"`
}

func parseDate(date string) time.Time {
	t, _ := time.Parse(time.DateOnly, date)
	return t
}
