package search

import (
	"context"
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Pool 管理一组可复用的 rod Page。
// 同一时刻一个 Page 只被一个 goroutine 使用，解决 rod 非并发安全的问题。
type Pool struct {
	pages   chan *rod.Page
	browser *rod.Browser
	size    int
}

// NewPool 连接浏览器并预分配 Page。
// browserURL 是 rod 浏览器的调试地址，如 "ws://127.0.0.1:9222"。
func NewPool(ctx context.Context, browserURL string, size int) (*Pool, error) {
	browser := rod.New().ControlURL(browserURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	p := &Pool{
		browser: browser,
		pages:   make(chan *rod.Page, size),
		size:    size,
	}

	for i := 0; i < size; i++ {
		page, err := browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return nil, fmt.Errorf("create page %d: %w", i, err)
		}
		p.pages <- page
	}

	log.Printf("[pool] %d pages ready", size)
	return p, nil
}

// Acquire 获取一个空闲 Page（阻塞直到有可用或 ctx 超时）。
func (p *Pool) Acquire(ctx context.Context) (*rod.Page, error) {
	select {
	case page := <-p.pages:
		return page, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release 归还 Page。如果 Page 已不可用则自动创建新的替补。
func (p *Pool) Release(page *rod.Page) {
	if page == nil {
		return
	}
	// 探测 page 是否存活
	_, err := page.Info()
	if err != nil {
		log.Printf("[pool] page dead, replacing...")
		newPage, createErr := p.browser.Page(proto.TargetCreateTarget{})
		if createErr != nil {
			log.Printf("[pool] failed to create replacement page: %v", createErr)
			return
		}
		p.pages <- newPage
		return
	}
	p.pages <- page
}

// Close 关闭浏览器和所有 Page。
func (p *Pool) Close() {
	close(p.pages)
	for page := range p.pages {
		if page != nil {
			page.Close()
		}
	}
	p.browser.Close()
	log.Println("[pool] closed")
}
