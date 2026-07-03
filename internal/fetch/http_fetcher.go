package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	scmodel "github.com/yuya-cpu/security-checks/model"
)

const maxFetchBody = 1 << 20 // 1MB

//
type HTTPFetcher struct {
	client *http.Client
}

func NewHTTPFetcher(client *http.Client) *HTTPFetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPFetcher{client: client}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, targetURL string) (*scmodel.HTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "web-security-proxy-scanner/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBody))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	finalURL := targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	return &scmodel.HTTPResponse{
		URL:        finalURL,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Cookies:    resp.Cookies(),
		Body:       string(body),
	}, nil
}
