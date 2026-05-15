package tmdb

// ── 领域模型（对外暴露） ──────────────────────────

// MediaType 区分电影/电视。
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// MediaItem 统一的影视条目，由 TMDB 收藏列表映射而来。
type MediaItem struct {
	TMDBID        int       `json:"tmdb_id"`
	Title         string    `json:"title"`
	OriginalTitle string    `json:"original_title"`
	Type          MediaType `json:"type"`
	Year          int       `json:"year"`

	// TV 特有
	Season  int `json:"season"`
	Episode int `json:"episode"`

	PosterURL string `json:"poster_url"`
	Overview  string `json:"overview"`
}

// Episode 某一集。
type Episode struct {
	SeasonNum  int    `json:"season_number"`
	EpisodeNum int    `json:"episode_number"`
	AirDate    string `json:"air_date"`
	Name       string `json:"name"`
}

// Season TV 季信息。
type Season struct {
	SeasonNum    int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	AirDate      string `json:"air_date"`
	Name         string `json:"name"`
}

// TVShow TV 剧集详情。
type TVShow struct {
	TMDBID    int      `json:"tmdb_id"`
	Name      string   `json:"name"`
	Seasons   []Season `json:"seasons"`
	PosterURL string   `json:"poster_url"`
	Overview  string   `json:"overview"`
	Year      int      `json:"year"`
}

// ── TMDB API 原始响应结构（内部使用） ──────────────

// apiWatchlistItem TMDB 收藏列表中的原始条目。
type apiWatchlistItem struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Name          string  `json:"name"`
	OriginalName  string  `json:"original_name"`
	MediaType     string  `json:"media_type"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	PosterPath    string  `json:"poster_path"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
	Popularity    float64 `json:"popularity"`
}

// apiWatchlistResponse 收藏列表 API 响应。
type apiWatchlistResponse struct {
	Page         int                `json:"page"`
	Results      []apiWatchlistItem `json:"results"`
	TotalPages   int                `json:"total_pages"`
	TotalResults int                `json:"total_results"`
}

// apiSeasonResponse TV 单季详情 API 响应。
type apiSeasonResponse struct {
	ID         int          `json:"id"`
	SeasonNum  int          `json:"season_number"`
	Name       string       `json:"name"`
	AirDate    string       `json:"air_date"`
	Episodes   []apiEpisode `json:"episodes"`
	PosterPath string       `json:"poster_path"`
}

// apiEpisode 单集原始数据。
type apiEpisode struct {
	SeasonNum   int     `json:"season_number"`
	EpisodeNum  int     `json:"episode_number"`
	AirDate     string  `json:"air_date"`
	Name        string  `json:"name"`
	Overview    string  `json:"overview"`
	VoteAverage float64 `json:"vote_average"`
	StillPath   string  `json:"still_path"`
}

// apiTVShowResponse TV 剧集详情 API 响应。
type apiTVShowResponse struct {
	ID              int                `json:"id"`
	Name            string             `json:"name"`
	OriginalName    string             `json:"original_name"`
	Overview        string             `json:"overview"`
	FirstAirDate    string             `json:"first_air_date"`
	PosterPath      string             `json:"poster_path"`
	NumberOfSeasons int                `json:"number_of_seasons"`
	Seasons         []apiSeasonSummary `json:"seasons"`
}

// apiSeasonSummary TV 详情中 seasons 数组的元素。
type apiSeasonSummary struct {
	SeasonNum    int    `json:"season_number"`
	Name         string `json:"name"`
	EpisodeCount int    `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}
