package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/lemonc7/silo/config"
)

const cookieFile = "bt_cookies.json"

// sessionCookies 登录后的关键 Cookie。
type sessionCookies struct {
	BrowserVerified string `json:"browser_verified"`
	PHPSessID       string `json:"PHPSESSID"`
	AppAuth         string `json:"app_auth"`
}

// BTClient 资源站客户端，使用 headless 浏览器通过 PoW 验证。
type BTClient struct {
	cfg     config.ResourceConfig
	debug   bool
	browser *rod.Browser
	cookies *sessionCookies
}

func NewBTClient(cfg config.ResourceConfig) *BTClient {
	return &BTClient{cfg: cfg}
}

// Debug 开启浏览器可视化，用于排查登录问题。
func (c *BTClient) Debug() *BTClient {
	c.debug = true
	return c
}

// Login 登录并缓存 Cookie。已缓存且有效则跳过登录。
func (c *BTClient) Login() error {
	if sc, err := c.loadCookies(); err == nil {
		c.cookies = sc
		if err := c.launchAndVerify(); err == nil {
			fmt.Println("[bt] 使用缓存 Cookie")
			return nil
		}
		fmt.Printf("[bt] 缓存失效: %v\n", err)
		c.browser.Close()
	}

	if err := c.launch(); err != nil {
		return fmt.Errorf("启动浏览器: %w", err)
	}

	sc, err := c.doLogin()
	if err != nil {
		c.browser.Close()
		return err
	}

	c.cookies = sc
	c.saveCookies(sc)
	fmt.Println("[bt] 登录成功")
	return nil
}

// Close 关闭浏览器。
func (c *BTClient) Close() {
	if c.browser != nil {
		c.browser.Close()
	}
}

// Search 搜索资源并解析磁力链接。
func (c *BTClient) Search(query string) ([]Torrent, error) {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("创建页面: %w", err)
	}
	defer page.Close()

	homeURL := "https://" + c.cfg.Domain
	if err := page.Navigate(homeURL); err != nil {
		return nil, fmt.Errorf("打开首页: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 搜索
	q, err := page.Element("#q")
	if err != nil {
		return nil, fmt.Errorf("搜索框: %w", err)
	}
	if err := q.Input(query); err != nil {
		return nil, fmt.Errorf("输入搜索词: %w", err)
	}

	btn, err := page.Element("#s_form > button")
	if err != nil {
		return nil, fmt.Errorf("搜索按钮: %w", err)
	}
	if err := btn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("点击搜索: %w", err)
	}

	wait := page.WaitOpen()

	time.Sleep(2 * time.Second)

	// 点击“剧集”分类标签（第 3 个 a）
	cat, err := page.Element("body > main > div.search_head > div.l > a:nth-child(3)")
	if err != nil {
		return nil, fmt.Errorf("分类标签: %w", err)
	}
	if err := cat.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("点击分类: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 点击第一个结果的图片 → 新标签页打开详情
	firstImg, err := page.Element("body > main > div.sr_lists > div > div.img > a")
	if err != nil {
		return nil, fmt.Errorf("搜索结果: %w", err)
	}
	if err := firstImg.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("打开详情: %w", err)
	}

	// 获取新标签页
	newPage, err := wait()
	if err != nil {
		return nil, fmt.Errorf("等待详情页: %w", err)
	}
	defer newPage.Close()

	time.Sleep(2 * time.Second)

	// 筛选“中字1080P”
	filter, err := newPage.ElementR("li", "中字1080P")
	if err == nil {
		filter.Click(proto.InputMouseButtonLeft, 1)
		time.Sleep(1 * time.Second)
	}

	// 解析磁力链接
	return c.extractTorrents(newPage)
}

// ── 磁力解析 ─────────────────────────────────────

func (c *BTClient) extractTorrents(page *rod.Page) ([]Torrent, error) {
	rows, err := page.Elements("table.bit_list tbody tr")
	if err != nil {
		return nil, fmt.Errorf("查询磁力列表: %w", err)
	}

	minSize := int64(c.cfg.MinSizeGB * 1024 * 1024 * 1024)
	var torrents []Torrent
	for _, row := range rows {
		link, err := row.Element("a.svg-tf")
		if err != nil {
			continue
		}
		title, err := link.Attribute("title")
		if err != nil || title == nil || *title == "" {
			continue
		}
		href, err := link.Attribute("href")
		if err != nil || href == nil || !strings.HasPrefix(*href, "magnet:?") {
			continue
		}
		sizeText := c.tdText(row, 3)
		seederText := c.tdText(row, 4)
		seeders, _ := strconv.Atoi(seederText)

		t := Torrent{
			Title:     *title,
			MagnetURL: *href,
			SizeBytes: parseSize(sizeText),
			Seeders:   seeders,
		}
		c.assignPriority(&t, minSize)
		torrents = append(torrents, t)
	}
	return torrents, nil
}

func (c *BTClient) tdText(row *rod.Element, n int) string {
	td, err := row.Element("td:nth-child(" + strconv.Itoa(n) + ")")
	if err != nil {
		return ""
	}
	text, err := td.Text()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

func parseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	re := regexp.MustCompile(`^([\d.]+)\s*([KMGTP]?B)$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0
	}
	val, _ := strconv.ParseFloat(matches[1], 64)
	switch matches[2] {
	case "KB":
		return int64(val * 1024)
	case "MB":
		return int64(val * 1024 * 1024)
	case "GB":
		return int64(val * 1024 * 1024 * 1024)
	case "TB":
		return int64(val * 1024 * 1024 * 1024 * 1024)
	}
	return int64(val)
}

func (c *BTClient) assignPriority(t *Torrent, minSize int64) {
	low := strings.ToLower(t.Title)
	base := resolutionPriority(low)

	labelOK := strings.Contains(low, "中字")
	sizeOK := t.SizeBytes >= minSize

	var offset int
	switch {
	case labelOK && sizeOK:
		offset = 0
	case labelOK:
		offset = 5
	case sizeOK:
		offset = 10
	default:
		offset = 15
	}
	t.Priority = base + offset
}

func resolutionPriority(title string) int {
	is2160 := strings.Contains(title, "2160") || strings.Contains(title, "4k")
	is1080 := strings.Contains(title, "1080")
	is60fps := strings.Contains(title, "60fps") || strings.Contains(title, "60帧")
	switch {
	case is2160 && is60fps:
		return 0
	case is2160:
		return 1
	case is1080 && is60fps:
		return 2
	case is1080:
		return 3
	default:
		return 5
	}
}

// ── 浏览器 ───────────────────────────────────────

func (c *BTClient) launch() error {
	l := launcher.New().
		Set("no-sandbox").
		Set("disable-dev-shm-usage").
		Set("disable-gpu").
		Set("disable-blink-features", "AutomationControlled")

	if !c.debug {
		l = l.Headless(true)
	}

	if c.inContainer() {
		l = l.Set("disable-dev-shm-usage")
	}

	url, err := l.Launch()
	if err != nil {
		return err
	}

	c.browser = rod.New().ControlURL(url).MustConnect()
	return nil
}

func (c *BTClient) launchAndVerify() error {
	if err := c.launch(); err != nil {
		return err
	}
	c.injectCookies()

	page, err := c.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return err
	}
	defer page.Close()

	if !c.verifyLogin(page) {
		return fmt.Errorf("登录状态失效")
	}
	return nil
}

// inContainer 简易检测是否在容器内运行。
func (c *BTClient) inContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

// ── 登录流程 ─────────────────────────────────────

func (c *BTClient) doLogin() (*sessionCookies, error) {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("创建页面: %w", err)
	}
	defer page.Close()

	homeURL := "https://" + c.cfg.Domain
	loginURL := homeURL + "/user/login"

	// 1. 访问首页通过 PoW
	if err := page.Navigate(homeURL); err != nil {
		return nil, fmt.Errorf("访问首页: %w", err)
	}
	time.Sleep(3 * time.Second)
	if _, err := page.Element("header"); err != nil {
		return nil, fmt.Errorf("首页 header 未出现: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 2. 登录页
	if err := page.Navigate(loginURL); err != nil {
		return nil, fmt.Errorf("访问登录页: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 3. 等待 PoW（browser_verified cookie）
	if err := c.waitCookie(page, "browser_verified", 20*time.Second); err != nil {
		return nil, fmt.Errorf("PoW 验证: %w", err)
	}

	// 4. 关闭弹窗
	_, _ = page.Eval(`() => {
		document.querySelectorAll('button').forEach(b => {
			if(b.innerText.includes('不再提醒') || b.innerText.includes('关闭')) b.click();
		});
	}`)
	time.Sleep(500 * time.Millisecond)

	// 5. 填写表单
	usernameEl, err := page.Element(`input[name="username"]`)
	if err != nil {
		return nil, fmt.Errorf("用户名输入框: %w", err)
	}
	if err := usernameEl.Input(c.cfg.Username); err != nil {
		return nil, fmt.Errorf("输入用户名: %w", err)
	}

	passwordEl, err := page.Element(`input[name="password"]`)
	if err != nil {
		return nil, fmt.Errorf("密码输入框: %w", err)
	}
	if err := passwordEl.Input(c.cfg.Password); err != nil {
		return nil, fmt.Errorf("输入密码: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 6. 点击登录
	btn, err := page.ElementR("button", "登录")
	if err != nil {
		return nil, fmt.Errorf("登录按钮: %w", err)
	}
	if err := btn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("点击登录: %w", err)
	}

	// 7. 等待 app_auth
	if err := c.waitCookie(page, "app_auth", 15*time.Second); err != nil {
		return nil, fmt.Errorf("登录: %w", err)
	}

	sc := &sessionCookies{}
	cookies, err := page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("提取 Cookie: %w", err)
	}
	for _, ck := range cookies {
		switch ck.Name {
		case "browser_verified":
			sc.BrowserVerified = ck.Value
		case "PHPSESSID":
			sc.PHPSessID = ck.Value
		case "app_auth":
			sc.AppAuth = ck.Value
		}
	}
	if sc.AppAuth == "" {
		return nil, fmt.Errorf("未获取到 app_auth")
	}

	return sc, nil
}

func (c *BTClient) waitCookie(page *rod.Page, name string, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("等待 cookie %q 超时", name)
		default:
			cookies, err := page.Cookies(nil)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			for _, ck := range cookies {
				if ck.Name == name {
					return nil
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// ── Cookie 管理 ──────────────────────────────────

func (c *BTClient) injectCookies() {
	c.browser.SetCookies([]*proto.NetworkCookieParam{
		{Name: "browser_verified", Value: c.cookies.BrowserVerified, Domain: "." + c.cfg.Domain},
		{Name: "PHPSESSID", Value: c.cookies.PHPSessID, Domain: "." + c.cfg.Domain},
		{Name: "app_auth", Value: c.cookies.AppAuth, Domain: "." + c.cfg.Domain},
	})
}

func (c *BTClient) verifyLogin(page *rod.Page) bool {
	if err := page.Navigate("https://" + c.cfg.Domain); err != nil {
		return false
	}
	time.Sleep(3 * time.Second)

	for i := 0; i < 10; i++ {
		el, err := page.Element("header")
		if err == nil && el != nil {
			time.Sleep(500 * time.Millisecond)
			res, err := page.Eval(`() => document.querySelector('.user_menu') !== null`)
			if err == nil {
				return res.Value.Bool()
			}
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func (c *BTClient) loadCookies() (*sessionCookies, error) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return nil, err
	}
	var sc sessionCookies
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	if sc.AppAuth == "" {
		return nil, fmt.Errorf("cookie 不完整")
	}
	return &sc, nil
}

func (c *BTClient) saveCookies(sc *sessionCookies) {
	data, _ := json.MarshalIndent(sc, "", "  ")
	os.WriteFile(cookieFile, data, 0600)
}
