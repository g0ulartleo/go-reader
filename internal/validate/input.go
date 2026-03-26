package validate

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	DefaultTimeoutMS = 10000
	MaxTimeoutMS     = 15000
)

func Input(rawURL string, timeoutMS int) (string, int, error) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return "", 0, fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", 0, fmt.Errorf("url must be absolute")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", 0, fmt.Errorf("url scheme must be http or https")
	}

	if timeoutMS <= 0 {
		timeoutMS = DefaultTimeoutMS
	}
	if timeoutMS > MaxTimeoutMS {
		timeoutMS = MaxTimeoutMS
	}

	return parsed.String(), timeoutMS, nil
}

func TimeoutParam(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
