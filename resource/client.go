package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/lemonc7/silo/config"
)

const cookieFile = "bt_cookies.json"

type BTClient struct {
	cfg     config.ResourceConfig
	browser *rod.Browser
	debug   bool
}

func NewBTClient(cfg config.ResourceConfig) *BTClient {
	return &BTClient{cfg: cfg}
}

func (c *BTClient) EnsureSession(ctx context.Context) error {
	if err := c.launch(ctx); err != nil {
		return err
	}

	cookies, err := loadCookies()
	if err != nil {
		fmt.Printf("[bt] 加载 cookie 失败: %v\n", err)
		return c.fullLogin(ctx)
	}
	if len(cookies) == 0 {
		return c.fullLogin(ctx)
	}

	if err := c.setCookies(cookies); err != nil {
		return c.fullLogin(ctx)
	}

	if err := c.verifySession(ctx); err == nil {
		fmt.Printf("[bt] 登录成功")
		return nil
	}

	fmt.Println("[bt] cookie 可能失效，执行完整登录")
	return c.fullLogin(ctx)
}

func (c *BTClient) Close() {
	if c.browser != nil {
		c.browser.Close()
	}
}

func (c *BTClient) Debug() {
	c.debug = true
}

func (c *BTClient) launch(ctx context.Context) error {
	if c.browser != nil {
		return nil
	}
	l := launcher.New().
		Set("disable-gpu").
		Set("disable-blink-features", "AutomationControlled")

	if !c.debug {
		l = l.Headless(true)
	}

	// Container runtime usually needs these for chrome stability.
	if c.inContainer() {
		l = l.
			Set("no-sandbox").
			Set("disable-dev-shm-usage")
	}

	devURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("启动浏览器: %w", err)
	}

	b := rod.New().NoDefaultDevice().ControlURL(devURL).Context(ctx)
	if err := b.Connect(); err != nil {
		return fmt.Errorf("连接浏览器: %w", err)
	}

	c.browser = b
	return nil
}

func (c *BTClient) inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}
	return false
}

func (c *BTClient) fullLogin(ctx context.Context) error {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL + "/user/login"})
	if err != nil {
		return fmt.Errorf("打开登录页: %w", err)
	}
	defer page.Close()
	page = page.Context(ctx)

	usernameEl, err := page.Element(`input[name="username"]`)
	if err != nil {
		return fmt.Errorf("用户名输入框: %w", err)
	}
	if err := usernameEl.Input(c.cfg.Username); err != nil {
		return fmt.Errorf("输入用户名: %w", err)
	}

	passwordEl, err := page.Element(`input[name="password"]`)
	if err != nil {
		return fmt.Errorf("密码输入框: %w", err)
	}
	if err := passwordEl.Input(c.cfg.Password); err != nil {
		return fmt.Errorf("输入密码: %w", err)
	}

	loginBtn, err := page.Element("#button")
	if err != nil {
		return fmt.Errorf("登录按钮: %w", err)
	}
	if err := loginBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("点击登录: %w", err)
	}

	cookies, err := c.waitAuthCookie(ctx)
	if err != nil {
		return err
	}
	if err := c.setCookies(cookies); err != nil {
		return err
	}
	saveCookies(cookies)

	fmt.Println("[bt] 登录成功，cookie 已缓存")
	return nil
}

func (c *BTClient) waitAuthCookie(ctx context.Context) ([]*proto.NetworkCookie, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("等待 app_auth 取消: %w", ctx.Err())
		default:
		}

		cookies, err := c.browser.GetCookies()
		if err == nil && hasCookie(cookies, "app_auth") {
			return cookies, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (c *BTClient) setCookies(cookies []*proto.NetworkCookie) error {
	params := proto.CookiesToParams(cookies)
	if len(params) == 0 {
		return fmt.Errorf("设置的 cookie 是空的")
	}
	if err := c.browser.SetCookies(params); err != nil {
		return fmt.Errorf("设置 cookie: %w", err)
	}
	return nil
}

func (c *BTClient) verifySession(ctx context.Context) error {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL})
	if err != nil {
		return fmt.Errorf("打开主页验证: %w", err)
	}
	defer page.Close()
	page = page.Context(ctx)

	var state string
	_, err = page.
		Race().
		Element("#user_load > div").
		Handle(func(e *rod.Element) error {
			fmt.Println("[bt] 校验登录状态: 已登录")
			state = "logged_in"
			return nil
		}).
		Element("#user_load > a").
		Handle(func(e *rod.Element) error {
			fmt.Println("[bt] 校验登录状态: 未登录")
			state = "need_login"
			return nil
		}).
		Do()

	if err != nil {
		return fmt.Errorf("判定登录状态失败: %w", err)
	}

	if state == "need_login" {
		return fmt.Errorf("未登录")
	}
	if state != "logged_in" {
		return fmt.Errorf("登录状态未知: %s", state)
	}

	return nil
}

func loadCookies() ([]*proto.NetworkCookie, error) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return nil, err
	}
	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, err
	}
	return cookies, nil
}

func saveCookies(cookies []*proto.NetworkCookie) {
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		fmt.Printf("[bt] 解码 cookie: %v\n", err)
		return
	}
	if err := os.WriteFile(cookieFile, data, 0o600); err != nil {
		fmt.Printf("[bt] 写入 cookie 文件: %v\n", err)
	}
}

func hasCookie(cookies []*proto.NetworkCookie, name string) bool {
	for _, ck := range cookies {
		if ck.Name == name {
			return true
		}
	}
	return false
}
