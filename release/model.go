package release

import (
	"context"

	"github.com/lemonc7/silo/catalog"
)

type Provider interface {
	EnsureSession(ctx context.Context) error
	Resolve(ctx context.Context, item Media) (string, error)
	FetchReleases(ctx context.Context, item Resource) ([]Torrent, error)
}

type Media struct {
	Type  catalog.MediaType
	Title string
	Year  int
}

type Resource struct {
	Target   string
	Type     catalog.MediaType
	SeasonID *int64
}

type Torrent struct {
	Title      string `json:"title"`
	MagnetUrl  string `json:"magnet_url"`
	SizeBytes  int64  `json:"size_bytes"`
	Seeders    int64  `json:"seeders"`
	Resolution string `json:"resolution"`
}
