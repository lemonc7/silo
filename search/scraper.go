package search

import (
	"context"

	"github.com/go-rod/rod"
)

// ── 数据模型 ─────────────────────────────────────

// ResourceResult 资源站搜索结果中的一条。
type ResourceResult struct {
	Title     string `json:"title"`      // 资源标题（如 "Show.Name.S01E03.1080p"）
	DetailURL string `json:"detail_url"` // 详情页地址
	Size      string `json:"size"`       // 文件大小
	Seeders   int    `json:"seeders"`    // 做种数
}

// MagnetLink 详情页中提取的磁力链接。
type MagnetLink struct {
	Name    string `json:"name"`
	Magnet  string `json:"magnet"`
	Size    string `json:"size"`
	Seeders int    `json:"seeders"`
}

// ── Scraper 接口 ─────────────────────────────────

// Scraper 资源站爬取接口。
// 定义对资源站的所有操作。具体实现依赖资源站的页面结构。
type Scraper interface {
	// Login 登录资源站（如需要）。应在每次会话开始前调用。
	Login(ctx context.Context, page *rod.Page, username, password string) error

	// Search 根据关键词搜索资源。
	Search(ctx context.Context, page *rod.Page, keyword string) ([]ResourceResult, error)

	// FetchMagnets 进入资源详情页提取磁力链接列表。
	FetchMagnets(ctx context.Context, page *rod.Page, detailURL string) ([]MagnetLink, error)
}

// ── NoopScraper 空实现（占位） ──────────────────

// NoopScraper 是 Scraper 的空实现，用于开发阶段占位。
// 后续根据具体资源站替换为真实实现。
type NoopScraper struct {
	BaseURL string
}

func NewNoopScraper(baseURL string) *NoopScraper {
	return &NoopScraper{BaseURL: baseURL}
}

func (s *NoopScraper) Login(ctx context.Context, page *rod.Page, username, password string) error {
	_ = ctx
	_ = page
	_ = username
	_ = password
	// TODO: 实现登录逻辑
	return nil
}

func (s *NoopScraper) Search(ctx context.Context, page *rod.Page, keyword string) ([]ResourceResult, error) {
	_ = ctx
	_ = page
	_ = keyword
	// TODO: 实现搜索逻辑
	return nil, nil
}

func (s *NoopScraper) FetchMagnets(ctx context.Context, page *rod.Page, detailURL string) ([]MagnetLink, error) {
	_ = ctx
	_ = page
	_ = detailURL
	// TODO: 实现详情页解析
	return nil, nil
}
