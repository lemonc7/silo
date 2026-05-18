package resource

type Client interface {
	Login() error
	Search(query string) ([]Torrent, error)
}

type Torrent struct {
	Title      string `json:"title"`
	MagnetUrl  string `json:"magnet_url"`
	SizeBytes  int64  `json:"size_bytes"`
	Seeders    int64  `json:"seeders"`
	Resolution string `json:"resolution"`
}
