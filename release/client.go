package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/lemonc7/episodex"
	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/config"
)

const cookieFile = "bt_cookies.json"

var (
	ErrNoMatch = errors.New("资源不存在")
)

type BTClient struct {
	cfg     config.ResourceConfig
	browser *rod.Browser
	debug   bool
	re      *regexp.Regexp
}

var _ Provider = (*BTClient)(nil)

func NewBTClient(cfg config.ResourceConfig) *BTClient {
	resPattern := strings.Join(cfg.Profiles, "|")
	pattern := fmt.Sprintf(`^中字(%s)$`, resPattern)
	return &BTClient{
		cfg:   cfg,
		debug: false,
		re:    regexp.MustCompile(pattern),
	}
}

func (c *BTClient) EnsureSession(ctx context.Context) error {
	if err := c.launch(ctx); err != nil {
		return err
	}

	cookies, err := loadCookies()
	if err != nil {
		log.Printf("[bt] 加载 cookie 失败: %v\n", err)
		return c.fullLogin(ctx)
	}
	if len(cookies) == 0 {
		return c.fullLogin(ctx)
	}

	if err := c.setCookies(cookies); err != nil {
		return c.fullLogin(ctx)
	}

	if err := c.verifySession(ctx); err == nil {
		latestCookies, getErr := c.browser.GetCookies()
		if getErr != nil {
			log.Printf("[bt] 登录成功，但刷新 cookie 失败: %v\n", getErr)
			return nil
		}
		if len(latestCookies) > 0 {
			saveCookies(latestCookies)
		}
		log.Println("[bt] 登录成功，cookie 已更新")
		return nil
	}

	log.Println("[bt] cookie 可能失效，执行完整登录")
	return c.fullLogin(ctx)
}

func (c *BTClient) Resolve(ctx context.Context, item Media) (string, error) {
	var queryType int
	switch item.Type {
	case catalog.MediaTypeMovie:
		queryType = 1
	case catalog.MediaTypeTV:
		queryType = 2
	case catalog.MediaTypeAnime:
		queryType = 3
	}

	url := fmt.Sprintf("%s/search?q=%s+%d&type=%d&mode=1",
		c.cfg.URL, url.QueryEscape(item.Title), item.Year, queryType)

	page, err := c.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return "", fmt.Errorf("打开页面: %w", err)
	}
	defer page.Close()
	page = page.Context(ctx)

	var result string

	_, err = page.Race().
		Element("div.sr_lists:empty").
		Handle(func(e *rod.Element) error {
			return fmt.Errorf("%w: %s-%d", ErrNoMatch, item.Title, item.Year)
		}).
		Element("div.sr_lists > div > div.img > a").
		Handle(func(e *rod.Element) error {
			href, err := e.Attribute("href")
			if err != nil {
				return fmt.Errorf("读取href: %w", err)
			}
			if href == nil || *href == "" {
				return fmt.Errorf("href为空")
			}

			result = *href
			return nil
		}).
		Do()

	if err != nil {
		return "", fmt.Errorf("搜索结果失败: %w", err)
	}

	return result, nil
}

func (c *BTClient) FetchReleases(ctx context.Context, target string) ([]Torrent, error) {
	url := c.cfg.URL + target
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, fmt.Errorf("打开资源详情页: %w", err)
	}
	defer page.Close()
	page = page.Context(ctx)

	var torrents []Torrent
	popupButton, err := page.Element("div.popup-close")
	if err != nil {
		return nil, fmt.Errorf("弹窗关闭按钮: %w", err)
	}
	if err := popupButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("关闭弹窗: %w", err)
	}
	tabParent, err := page.Element("ul.nav-tabs")
	if err != nil {
		return nil, fmt.Errorf("资源标签页: %w", err)
	}
	tabs, err := tabParent.Elements("li")
	if err != nil {
		return nil, fmt.Errorf("资源标签页列表: %w", err)
	}
	// 无磁力链接提前返回
	if len(tabs) == 1 {
		return nil, fmt.Errorf("资源无(期望的)磁力链接")
	}

	for _, tab := range tabs {
		text, err := tab.Text()
		if err != nil {
			return nil, fmt.Errorf("标签页名称: %w", err)
		}
		profile := strings.Fields(text)[0]
		if !slices.Contains(c.cfg.Profiles, profile) {
			continue
		}

		if err := tab.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return nil, fmt.Errorf("点击标签页: %w", err)
		}
		table, err := page.Element("table.bit_list > tbody")
		if err != nil {
			return nil, fmt.Errorf("资源表格: %w", err)
		}
		rows, err := table.Elements("tr")
		if err != nil {
			return nil, fmt.Errorf("读取资源列表: %w", err)
		}

		ts, err := c.getTorrents(ctx, rows, profile)
		if err != nil {
			return nil, fmt.Errorf("获取磁力链接: %w", err)
		}
		torrents = append(torrents, ts...)
	}

	if len(torrents) == 0 {
		return nil, fmt.Errorf("无预期的磁力链接")
	}

	return torrents, nil
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

	log.Println("[bt] 登录成功，cookie 已缓存")
	return nil
}

func (c *BTClient) waitAuthCookie(ctx context.Context) ([]*proto.NetworkCookie, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("等待 app_auth 超时: %w", ctx.Err())
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
		return fmt.Errorf("设置的 cookie 为空")
	}
	if err := c.browser.SetCookies(params); err != nil {
		return fmt.Errorf("设置 cookie: %w", err)
	}
	return nil
}

func (c *BTClient) verifySession(ctx context.Context) error {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL})
	if err != nil {
		return fmt.Errorf("打开首页验证: %w", err)
	}
	defer page.Close()
	page = page.Context(ctx)

	var state string
	_, err = page.
		Race().
		Element("#user_load > div").
		Handle(func(e *rod.Element) error {
			log.Println("[bt] 校验登录状态: 已登录")
			state = "logged_in"
			return nil
		}).
		Element("#user_load > a").
		Handle(func(e *rod.Element) error {
			log.Println("[bt] 校验登录状态: 未登录")
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
		log.Printf("[bt] 编码 cookie: %v\n", err)
		return
	}
	if err := os.WriteFile(cookieFile, data, 0o600); err != nil {
		log.Printf("[bt] 写入 cookie 文件: %v\n", err)
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

func (c *BTClient) getTorrents(ctx context.Context, rows rod.Elements, profile string) ([]Torrent, error) {
	var ts []Torrent
	for _, row := range rows {
		visible, err := row.Visible()
		if err != nil {
			return nil, fmt.Errorf("元素可视性: %w", err)
		}
		if !visible {
			continue
		}

		nameEl, err := row.Element("a.svg-tf")
		if err != nil {
			return nil, fmt.Errorf("名称列: %w", err)
		}
		href, err := nameEl.Attribute("href")
		if err != nil {
			return nil, fmt.Errorf("读取磁力链接: %w", err)
		}
		if href == nil {
			return nil, fmt.Errorf("获取资源href: %w", err)
		}

		if strings.HasPrefix(*href, "magnet:") {
			title, err := nameEl.Text()
			if err != nil {
				return nil, fmt.Errorf("获取标题: %w", err)
			}

			sizeEl, err := row.Element("td:nth-child(3)")
			if err != nil {
				return nil, fmt.Errorf("大小列: %w", err)
			}
			size, err := sizeEl.Text()
			if err != nil {
				return nil, fmt.Errorf("获取资源大小: %w", err)
			}

			seederEl, err := row.Element("td:nth-child(4) > i")
			if err != nil {
				return nil, fmt.Errorf("做种列: %w", err)
			}
			seederStr, err := seederEl.Text()
			if err != nil {
				return nil, fmt.Errorf("获取做种数: %w", err)
			}
			seeder, _ := strconv.Atoi(seederStr)

			ts = append(ts, Torrent{
				Title:   title,
				Magnet:  *href,
				Size:    parseSizeMB(size),
				Seeder:  int64(seeder),
				Profile: profile,
			})
		} else if strings.HasPrefix(*href, "/bt") {
			detailPage, err := c.browser.Page(proto.TargetCreateTarget{URL: c.cfg.URL + *href})
			if err != nil {
				return nil, fmt.Errorf("打开种子详情页: %w", err)
			}
			defer detailPage.Close()
			detailPage = detailPage.Context(ctx)

			list, err := detailPage.Element("ul.down321")
			if err != nil {
				return nil, fmt.Errorf("种子列表: %w", err)
			}
			lis, err := list.Elements("li")
			for _, li := range lis {
				titleEl, err := li.Element("div:nth-child(1)")
				if err != nil {
					return nil, fmt.Errorf("种子标题列: %w", err)
				}
				title, err := titleEl.Text()
				if err != nil {
					return nil, fmt.Errorf("获取标题: %w", err)
				}

				magnetEl, err := li.Element(`a[href^="magnet:"]`)
				if err != nil {
					return nil, fmt.Errorf("磁力链接: %w", err)
				}
				magnet, err := magnetEl.Attribute("href")
				if err != nil {
					return nil, fmt.Errorf("获取磁力链接: %w", err)
				}

				sizeEl, err := li.Element("div.left")
				if err != nil {
					return nil, fmt.Errorf("种子大小: %w", err)
				}
				sizeStr, err := sizeEl.Text()
				if err != nil {
					return nil, fmt.Errorf("获取种子大小: %w", err)
				}

				ep := episodex.ExtractEpisodeInfo(title)
				var episodes []int64
				if ep.Start != nil && ep.End != nil {
					for i := ep.Start.Major; i <= ep.End.Major; i++ {
						episodes = append(episodes, int64(i))
					}
				} else if ep.Episode != nil {
					episodes = append(episodes, int64(ep.Episode.Major))
				}
				ts = append(ts, Torrent{
					Title:    title,
					Magnet:   *magnet,
					Size:     parseSizeMB(sizeStr),
					Seeder:   0,
					Profile:  profile,
					Episodes: episodes,
				})
			}
		} else {
			log.Printf("[bt] 无效的href类型: %s\n", *href)
		}

	}

	return ts, nil
}

var sizeRe = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*([kmgt]?b)`)

func parseSizeMB(raw string) float64 {
	matches := sizeRe.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(matches[2]) {
	case "KB":
		return value / 1024
	case "MB":
		return value
	case "GB":
		return value * 1024
	case "TB":
		return value * 1024 * 1024
	default:
		return 0
	}
}
