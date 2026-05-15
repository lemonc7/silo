package download

import (
	"context"
	"time"
)

// ── 数据模型 ─────────────────────────────────────

// TaskStatus 下载任务状态。
type TaskStatus string

const (
	StatusPending     TaskStatus = "pending"
	StatusDownloading TaskStatus = "downloading"
	StatusCompleted   TaskStatus = "completed"
	StatusFailed      TaskStatus = "failed"
)

// Task 一个下载任务。
type Task struct {
	ID        string     `json:"id"`
	Magnet    string     `json:"magnet"`
	Dir       string     `json:"dir"`
	Status    TaskStatus `json:"status"`
	Progress  float64    `json:"progress"` // 0.0 ~ 1.0
	CreatedAt time.Time  `json:"created_at"`
}

// ── Downloader 接口 ──────────────────────────────

// Downloader 下载管理接口。
// 定义添加任务、查询状态的能力。具体实现可以是 aria2 / transmission / qBittorrent 等。
type Downloader interface {
	// AddMagnet 添加磁力链接，返回任务 ID。
	AddMagnet(ctx context.Context, magnet string, dir string) (taskID string, err error)

	// GetStatus 查询任务状态。
	GetStatus(ctx context.Context, taskID string) (*Task, error)

	// Remove 删除任务（含文件）。
	Remove(ctx context.Context, taskID string) error
}

// ── NoopDownloader 空实现（占位） ────────────────

// NoopDownloader 是 Downloader 的空实现，用于开发阶段占位。
type NoopDownloader struct{}

func NewNoopDownloader() *NoopDownloader {
	return &NoopDownloader{}
}

func (d *NoopDownloader) AddMagnet(ctx context.Context, magnet string, dir string) (string, error) {
	// TODO: 实现下载器接入（aria2 / transmission / qBittorrent）
	_ = ctx
	_ = magnet
	_ = dir
	return "", nil
}

func (d *NoopDownloader) GetStatus(ctx context.Context, taskID string) (*Task, error) {
	_ = ctx
	_ = taskID
	return nil, nil
}

func (d *NoopDownloader) Remove(ctx context.Context, taskID string) error {
	_ = ctx
	_ = taskID
	return nil
}
