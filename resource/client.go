package resource

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/lemonc7/silo/config"
)

const cookieFile = "bt_cookies.json"

type sessionCookies struct {
	BrowserVerified string `json:"browser_verified"`
	PHPSessID       string `json:"PHPSESSID"`
	AppAuth         string `json:"app_auth"`
}

type BTClient struct {
	cfg     config.ResourceConfig
	debug   bool
	browser *rod.Browser
	cookies *sessionCookies
}

func NewBTClient(cfg config.ResourceConfig) *BTClient {
	return &BTClient{cfg: cfg}
}

func (c *BTClient) Debug() *BTClient {
	c.debug = true
	return c
}

func (c *BTClient) Login() error {
	// if sc, err := c.loadCookies(); err == nil {
	// 	c.cookies = sc
	// 	if err := c.launchAndVerify(); err == nil {
	// 		fmt.Println("[bt] 使用缓存 Cookie")
	// 		return nil
	// 	}
	// 	fmt.Printf("[bt] 缓存失效: %v\n", err)
	// 	c.browser.Close()
	// }

	if err := c.launch(); err != nil {
		return fmt.Errorf("启动浏览器: %w", err)
	}

	_, err := c.doLogin()
	if err != nil {
		c.Close()
		return err
	}

	// c.cookies = sc
	// c.saveCookies(sc)
	fmt.Println("[bt] 登录成功")
	return nil
}

func (c *BTClient) Close() {
	if c.browser != nil {
		c.browser.Close()
	}
}

func (c *BTClient) launch() error {
	l := launcher.New()

	devURL, err := l.Launch()
	if err != nil {
		return err
	}

	c.browser = rod.New().NoDefaultDevice().ControlURL(devURL)
	return c.browser.Connect()
}

func (c *BTClient) doLogin() (*sessionCookies, error) {
	page, err := c.browser.Page(proto.TargetCreateTarget{
		URL: c.cfg.URL + "/user/login",
	})
	if err != nil {
		return nil, fmt.Errorf("打开登录页: %w", err)
	}
	defer page.Close()

	// 填写表单
	usernameEl, err := page.ElementR("input", "请输入用户名")
	if err != nil {
		return nil, fmt.Errorf("获取用户名输入框: %w", err)
	}
	if err := usernameEl.Input(c.cfg.Username); err != nil {
		return nil, fmt.Errorf("输入用户名: %w", err)
	}

	passwordEl, err := page.ElementR("input", "请输入密码")
	if err != nil {
		return nil, fmt.Errorf("获取密码输入框: %w", err)
	}
	if err := passwordEl.Input(c.cfg.Password); err != nil {
		return nil, fmt.Errorf("输入密码: %w", err)
	}

	// 点击登录
	loginBtn, err := page.Element("#button")
	if err != nil {
		return nil, fmt.Errorf("获取登录按钮: %w", err)
	}
	if err := loginBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("点击登录: %w", err)
	}

	return nil, nil
}
