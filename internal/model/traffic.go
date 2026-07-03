package model

import "time"

// HTTPTransaction は1回のHTTP通信（リクエスト＋レスポンス）を表す。
type HTTPTransaction struct {
	ID           int64     `json:"id"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	RequestHeaders  string `json:"request_headers"`
	RequestBody     string `json:"request_body"`
	StatusCode   int       `json:"status_code"`
	ResponseHeaders string `json:"response_headers"`
	ResponseBody    string `json:"response_body"`
	DurationMS   int64     `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}
