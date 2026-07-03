package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/repository"
)

const repeaterMaxBodySize = 1 << 20 // 1MB

//
type RepeaterService struct {
	repo   repository.TrafficRepository
	client *http.Client
}

func NewRepeaterService(repo repository.TrafficRepository, client *http.Client) *RepeaterService {
	if client == nil {
		client = &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &RepeaterService{repo: repo, client: client}
}

//
func (s *RepeaterService) Send(ctx context.Context, input model.RepeaterRequest) (*model.HTTPTransaction, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		return nil, fmt.Errorf("method is required")
	}
	if method == http.MethodConnect {
		return nil, fmt.Errorf("CONNECT method is not supported in repeater")
	}

	url := strings.TrimSpace(input.URL)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	headers, err := parseHeaderText(input.Headers)
	if err != nil {
		return nil, err
	}

	bodyReader := strings.NewReader(input.Body)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header = headers

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		tx := &model.HTTPTransaction{
			Method:          method,
			URL:             url,
			RequestHeaders:  formatHeaderText(headers),
			RequestBody:     input.Body,
			StatusCode:      http.StatusBadGateway,
			ResponseHeaders: "X-Repeater-Error: true",
			ResponseBody:    err.Error(),
			DurationMS:      time.Since(start).Milliseconds(),
			CreatedAt:       time.Now().UTC(),
		}
		id, saveErr := s.repo.Save(ctx, tx)
		if saveErr != nil {
			return nil, fmt.Errorf("send failed: %w; save failed: %v", err, saveErr)
		}
		tx.ID = id
		return tx, nil
	}
	defer resp.Body.Close()

	respBody, err := readLimitedBody(resp.Body, repeaterMaxBodySize)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	tx := &model.HTTPTransaction{
		Method:          method,
		URL:             url,
		RequestHeaders:  formatHeaderText(req.Header),
		RequestBody:     input.Body,
		StatusCode:      resp.StatusCode,
		ResponseHeaders: formatHeaderText(resp.Header),
		ResponseBody:    string(respBody),
		DurationMS:      time.Since(start).Milliseconds(),
		CreatedAt:       time.Now().UTC(),
	}

	id, err := s.repo.Save(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}
	tx.ID = id
	return tx, nil
}

func parseHeaderText(raw string) (http.Header, error) {
	headers := http.Header{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header line: %q", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid header line: %q", line)
		}
		headers.Add(key, value)
	}
	return headers, nil
}

func formatHeaderText(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := headers.Write(&buf); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func readLimitedBody(body io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(body, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return data[:limit], nil
	}
	return data, nil
}
