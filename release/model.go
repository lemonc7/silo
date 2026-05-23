package release

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

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
	Title   string  `json:"title"`
	Magnet  string  `json:"magnet"`
	Size    float64 `json:"size"`
	Seeder  int64   `json:"seeder"`
	Profile string  `json:"profile"`
}

func (t Torrent) String() string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(t); err != nil {
		return ""
	}

	return strings.TrimSpace(buf.String())
}
