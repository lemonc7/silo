package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lemonc7/silo/config"
)

type QBClient struct {
	baseURL  string
	client   *http.Client
	username string
	password string
	sid      string
}

func NewQBClient(cfg config.DownloaderConfig) *QBClient {
	return &QBClient{
		baseURL:  fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port),
		client:   &http.Client{},
		username: cfg.Username,
		password: cfg.Password,
	}
}

func (q *QBClient) Login() error {
	data := url.Values{}
	data.Set("username", q.username)
	data.Set("password", q.password)

	req, err := http.NewRequest("POST", q.baseURL+"/api/v2/auth/login", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Referer", q.baseURL+"/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("qb login: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "Ok." {
		return fmt.Errorf("response error: %s", string(body))
	}

	for _, c := range resp.Cookies() {
		if c.Name == "SID" {
			q.sid = c.Value
		}
	}
	return nil
}

func (q *QBClient) AddMagnet(ctx context.Context, magnetURL, savePath string) error {
	data := url.Values{}
	data.Set("urls", magnetURL)
	data.Set("savepath", savePath)

	req, err := http.NewRequestWithContext(ctx, "POST", q.baseURL+"/api/v2/torrents/add", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if q.sid != "" {
		req.Header.Set("Cookie", "SID="+q.sid)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("qB add magnet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qB add magnet: %s", resp.Status)
	}
	return nil
}
