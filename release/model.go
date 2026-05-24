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
	FetchReleases(ctx context.Context, target string) ([]Torrent, error)
}

type Media struct {
	Type  catalog.MediaType
	Title string
	Year  int
}

type Torrent struct {
	Title    string  `json:"title"`
	Magnet   string  `json:"magnet"`
	Size     float64 `json:"size"`
	Seeder   int64   `json:"seeder"`
	Profile  string  `json:"profile"`
	Episodes []int64 `json:"episodes"`
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
