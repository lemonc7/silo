package download

import "context"

type Downloader interface {
	AddMagnet(ctx context.Context, magnetURL, savePath string) error
}
