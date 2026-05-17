package download

import "context"

// Client 下载器接口（qBittorrent / Aria2 等）。
type Client interface {
	AddMagnet(ctx context.Context, magnetURL, savePath string) error
}
