package resource

import (
	"context"

	"github.com/lemonc7/silo/media"
)

type Resource interface {
	EnsureSession(ctx context.Context) error
	Resolve(ctx context.Context, item Media) (string, error)
	FetchReleases(url string) ([]Torrent, error)
}

type Media struct {
	Type  media.MediaType
	Title string
	Year  int
}

type Torrent struct {
	Title      string `json:"title"`
	MagnetUrl  string `json:"magnet_url"`
	SizeBytes  int64  `json:"size_bytes"`
	Seeders    int64  `json:"seeders"`
	Resolution string `json:"resolution"`
}
