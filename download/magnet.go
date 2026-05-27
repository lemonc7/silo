package download

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

func ExtractInfoHash(magnetURL string) (string, error) {
	u, err := url.Parse(magnetURL)
	if err != nil {
		return "", fmt.Errorf("解析磁力链接: %w", err)
	}
	if u.Scheme != "magnet" {
		return "", fmt.Errorf("不是磁力链接: %s", u.Scheme)
	}

	for _, xt := range u.Query()["xt"] {
		raw, ok := strings.CutPrefix(strings.ToLower(xt), "urn:btih:")
		if !ok {
			continue
		}
		if len(raw) == 40 {
			if _, err := hex.DecodeString(raw); err != nil {
				return "", fmt.Errorf("非法 hex info hash: %w", err)
			}
			return raw, nil
		}
		if len(raw) == 32 {
			data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(raw))
			if err != nil {
				return "", fmt.Errorf("非法 base32 info hash: %w", err)
			}
			if len(data) != 20 {
				return "", fmt.Errorf("非法 base32 info hash 长度: %d", len(data))
			}
			return hex.EncodeToString(data), nil
		}
		return "", fmt.Errorf("不支持的 info hash 长度: %d", len(raw))
	}

	return "", fmt.Errorf("磁力链接缺少 xt=urn:btih")
}
