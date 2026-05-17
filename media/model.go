package media

import (
	"encoding/json"
	"slices"
)

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
)

type MediaItem struct {
	TMDBID     int64       `json:"tmdb_id"`
	Title      string    `json:"title"`
	Type       MediaType `json:"type"`
	AirDate    string    `json:"air_date"`
	PosterPath string    `json:"poster_path"`
}

func (m MediaItem) String() string {
	jsonData, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonData)
}

type rawItem struct {
	ID           int64    `json:"id"`
	Title        string `json:"title"`
	Name         string `json:"name"`
	ReleaseDate  string `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
	PosterPath   string `json:"poster_path"`
	GenreIDs     []int  `json:"genre_ids"`
}

func (m *MediaItem) UnmarshalJSON(data []byte) error {
	var r rawItem
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.TMDBID = r.ID
	m.PosterPath = r.PosterPath

	if r.ReleaseDate != "" {
		m.AirDate = r.ReleaseDate
	} else {
		m.AirDate = r.FirstAirDate
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

type mediaResponse struct {
	Page         int         `json:"page"`
	Results      []MediaItem `json:"results"`
	TotalPages   int         `json:"total_pages"`
	TotalResults int         `json:"total_results"`
}

type seasonResponse struct {
	Seasons []seasonRaw `json:"seasons"`
}

type seasonRaw struct {
	SeasonNumber int64  `json:"season_number"`
	EpisodeCount int64  `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type episodesResponse struct {
	Episodes []episodeRaw `json:"episodes"`
}

type episodeRaw struct {
	EpisodeNumber int64  `json:"episode_number"`
	AirDate       string `json:"air_date"`
}

type Season struct {
	SeriesID     int64  `json:"series_id"`
	SeasonNumber int64  `json:"season_number"`
	EpisodeCount int64  `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type Episode struct {
	SeasonID      int64  `json:"season_id"`
	EpisodeNumber int64  `json:"episode_number"`
	AirDate       string `json:"air_date"`
	Status        string `json:"status"`
}
