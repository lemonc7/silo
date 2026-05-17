package resource

import "encoding/json"

// Torrent 一条磁力链接资源。
type Torrent struct {
	Title     string `json:"title"`
	MagnetURL string `json:"magnet_url"`
	SizeBytes int64  `json:"size_bytes"`
	Seeders   int    `json:"seeders"`
	Priority  int    `json:"priority"`
}

func (t Torrent) String() string {
	jsonData, err := json.MarshalIndent(&t, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// Client 资源站接口（后续每个站一个实现）。
type Client interface {
	Login() error
	Search(query string) ([]Torrent, error)
}
