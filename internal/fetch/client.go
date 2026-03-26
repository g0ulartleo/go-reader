package fetch

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxBodyBytes = 5 * 1024 * 1024
const defaultUserAgent = "go-reader/0.1"

type Client struct {
	http *http.Client
}

type Result struct {
	Body        string
	FinalURL    string
	StatusCode  int
	ContentType string
}

func NewClient() *Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Client{http: &http.Client{Transport: transport}}
}

func (c *Client) Get(ctx context.Context, rawURL string, userAgent string) (Result, error) {
	var lastErr error
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return Result{}, fmt.Errorf("bad url: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt == 0 && retryable(err) {
				time.Sleep(150 * time.Millisecond)
				continue
			}
			return Result{}, err
		}

		body, readErr := readBody(resp.Body, maxBodyBytes)
		_ = resp.Body.Close()
		if readErr != nil {
			return Result{StatusCode: resp.StatusCode, ContentType: resp.Header.Get("Content-Type")}, readErr
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 && attempt == 0 {
			lastErr = fmt.Errorf("upstream %d", resp.StatusCode)
			time.Sleep(150 * time.Millisecond)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			return Result{
				FinalURL:    resp.Request.URL.String(),
				StatusCode:  resp.StatusCode,
				ContentType: resp.Header.Get("Content-Type"),
			}, fmt.Errorf("upstream status %d", resp.StatusCode)
		}

		return Result{
			Body:        body,
			FinalURL:    resp.Request.URL.String(),
			StatusCode:  resp.StatusCode,
			ContentType: resp.Header.Get("Content-Type"),
		}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("fetch failed")
	}
	return Result{}, lastErr
}

func readBody(body io.Reader, limit int64) (string, error) {
	limited := io.LimitReader(body, limit+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(b)) > limit {
		return "", fmt.Errorf("response too large")
	}
	return string(b), nil
}

func retryable(err error) bool {
	if timeout(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe") || strings.Contains(msg, "refused")
}

func timeout(err error) bool {
	type timeout interface{ Timeout() bool }
	if t, ok := err.(timeout); ok && t.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded")
}

func IsTimeout(err error) bool {
	return timeout(err)
}
