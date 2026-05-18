package resource

import (
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
}

func NewBTClient(cfg config.ResourceConfig) *BTClient {
	return &BTClient{cfg: cfg}
}

func (c *BTClient) Login() error {
	l := launcher.New()
	devUrl, err := l.Launch()
	if err != nil {
		return fmt.Errorf("启动浏览器: %w", err)
	}
	c.browser = rod.New().NoDefaultDevice().ControlURL(devUrl)
	if err := c.browser.Connect(); err != nil {
		return fmt.Errorf("连接浏览器: %w", err)
	}

	cookies, err := c.loadCookies()
	if err != nil {
		fmt.Printf("[bt] 加载 cookie 失败: %v\n", err)
	}
	if len(cookies) == 0 {
		return c.fullLogin()
	}

	if err := c.setCookies(cookies); err != nil {
		return c.fullLogin()
	}

	if c.isExpired(cookies, "app_auth") {
		fmt.Println("[bt] app_auth 过期，重新登录")
		return c.fullLogin()
	}
	if c.isExpired(cookies, "browser_verified") {
		fmt.Println("[bt] browser_verified 过期，刷新 POW")
		return c.refreshPOW()
	}

	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL})
	if err != nil {
		return fmt.Errorf("验证登录: %w", err)
	}
	defer page.Close()

	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("等待主页加载完成: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("页面信息: %w", err)
	}

	if info.Title != "首页" {
		fmt.Printf("[bt] 标题不对: %s, 也许 cookie 已经失效，重新登录", info.Title)
		return c.fullLogin()
	}

	fmt.Printf("[bt] 登录成功")
	return nil
}

func (c *BTClient) Search(query string) ([]Torrent, error) {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL})
	if err != nil {
		return nil, fmt.Errorf("打开主页: %w", err)
	}
	defer page.Close()

	popupButton, err := page.Element("div.popup-close")
	if err != nil {
		return nil, fmt.Errorf("弹窗按钮: %w", err)
	}
	if err := popupButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("关闭弹窗: %w", err)
	}

	search, err := page.Element("#q")
	if err != nil {
		return nil, fmt.Errorf("搜索框: %w", err)
	}
	if err := search.Input(query); err != nil {
		return nil, fmt.Errorf("输入搜索词: %w", err)
	}

	button, err := page.Element("#s_form > button")
	if err != nil {
		return nil, fmt.Errorf("搜索按钮: %w", err)
	}
	if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("点击搜索: %w", err)
	}

	return nil, nil
}

func (c *BTClient) Close() {
	if c.browser != nil {
		c.browser.Close()
	}
}

func (c *BTClient) loadCookies() ([]*proto.NetworkCookie, error) {
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

func (c *BTClient) saveCookies(cookies []*proto.NetworkCookie) {
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		fmt.Printf("[bt] 解码 cookie: %v\n", err)
		return
	}
	if err := os.WriteFile(cookieFile, data, 0o600); err != nil {
		fmt.Printf("[bt] 写入 cookie 文件: %v\n", err)
	}
}

func (c *BTClient) isExpired(cookies []*proto.NetworkCookie, name string) bool {
	for _, ck := range cookies {
		if ck.Name == name && ck.Expires > 0 && int64(ck.Expires) < time.Now().Unix() {
			return true
		}
	}
	return false
}

func (c *BTClient) setCookies(cookies []*proto.NetworkCookie) error {
	var params []*proto.NetworkCookieParam
	for _, ck := range cookies {
		params = append(params, &proto.NetworkCookieParam{
			Name:   ck.Name,
			Value:  ck.Value,
			Domain: ck.Domain,
		})
	}
	if len(params) == 0 {
		return fmt.Errorf("设置的 cookie 是空的")
	}
	if err := c.browser.SetCookies(params); err != nil {
		return fmt.Errorf("设置 cookie: %w", err)
	}
	return nil
}

func (c *BTClient) fullLogin() error {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL + "/user/login"})
	if err != nil {
		return fmt.Errorf("打开登录页: %w", err)
	}
	defer page.Close()

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

	cookies, err := page.Cookies(nil)
	if err != nil {
		return fmt.Errorf("获取 cookie: %w", err)
	}

	if err := c.setCookies(cookies); err != nil {
		return err
	}
	c.saveCookies(cookies)

	fmt.Println("[bt] 登录成功，cookie 已缓存")
	return nil
}

func (c *BTClient) refreshPOW() error {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL})
	if err != nil {
		return err
	}
	defer page.Close()

	cookies, err := page.Cookies(nil)
	if err != nil {
		return err
	}

	if err := c.setCookies(cookies); err != nil {
		return err
	}

	c.saveCookies(cookies)
	fmt.Println("[bt] POW 刷新完成")
	return nil
}
