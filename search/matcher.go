package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lemonc7/silo/tmdb"
)

// Matcher 负责将 TMDB 条目匹配到资源站的搜索结果。
type Matcher struct{}

// NewMatcher 创建匹配器。
func NewMatcher() *Matcher {
	return &Matcher{}
}

// BestMatch 从搜索结果中找出最匹配 TMDB 条目的那一条。
// 对于电影：匹配标题 + 年份
// 对于剧集：匹配标题 + 季/集格式（S01E03、1x03 等）
//
// 返回 nil 表示没有匹配项。
func (m *Matcher) BestMatch(item tmdb.MediaItem, results []ResourceResult) *ResourceResult {
	for i := range results {
		if m.isMatch(item, results[i].Title) {
			return &results[i]
		}
	}
	return nil
}

// isMatch 判断资源标题是否匹配 TMDB 条目。
func (m *Matcher) isMatch(item tmdb.MediaItem, resourceTitle string) bool {
	rTitle := strings.ToLower(resourceTitle)
	tTitle := strings.ToLower(item.Title)

	// 标题必须包含
	if !strings.Contains(rTitle, tTitle) {
		return false
	}

	if item.Type == tmdb.MediaTypeTV {
		// 匹配季/集格式
		patterns := []string{
			fmt.Sprintf(`s%02de%02d`, item.Season, item.Episode),              // S01E03
			fmt.Sprintf(`%dx%02d`, item.Season, item.Episode),                 // 1x03
			fmt.Sprintf(`第\s*%d\s*季.*?第\s*%d\s*集`, item.Season, item.Episode), // 第1季第3集
		}
		for _, p := range patterns {
			if match, _ := regexp.MatchString(p, rTitle); match {
				return true
			}
		}
		return false
	}

	// 电影：额外匹配年份
	if item.Year > 0 {
		return strings.Contains(rTitle, fmt.Sprintf("%d", item.Year))
	}
	return true
}

// GenerateSearchKeyword 根据 TMDB 条目生成最优搜索关键词。
func (m *Matcher) GenerateSearchKeyword(item tmdb.MediaItem) string {
	if item.Type == tmdb.MediaTypeTV {
		return fmt.Sprintf("%s S%02dE%02d", item.Title, item.Season, item.Episode)
	}
	return fmt.Sprintf("%s %d", item.Title, item.Year)
}
