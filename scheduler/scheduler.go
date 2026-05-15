package scheduler

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器，封装 cron 库。
type Scheduler struct {
	cron *cron.Cron
}

// New 创建调度器实例。
func New() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
	}
}

// AddFunc 注册一个定时任务。
// spec: cron 表达式，如 "0 */6 * * *"（每 6 小时）。
func (s *Scheduler) AddFunc(spec string, fn func()) error {
	_, err := s.cron.AddFunc(spec, fn)
	if err != nil {
		return err
	}
	log.Printf("[scheduler] registered: %s", spec)
	return nil
}

// Start 启动调度器（非阻塞，在后台运行）。
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("[scheduler] started")
}

// Stop 停止调度器，返回可等待的 context。
func (s *Scheduler) Stop() context.Context {
	log.Println("[scheduler] stopping...")
	return s.cron.Stop()
}
