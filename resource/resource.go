package resource

import (
	"context"
)

type Resource interface {
	EnsureSession(ctx context.Context) error
	Resolve(ctx context.Context, media Media) (string, error)
	FetchReleases(url string) ([]Torrent, error)
}

type Media struct {
	Type   string
	Title  string
	Year   int
	Season *int
}

type Torrent struct {
	Title      string `json:"title"`
	MagnetUrl  string `json:"magnet_url"`
	SizeBytes  int64  `json:"size_bytes"`
	Seeders    int64  `json:"seeders"`
	Resolution string `json:"resolution"`
}
