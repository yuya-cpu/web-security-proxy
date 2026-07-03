package model

// RepeaterRequest は Repeater から再送信するリクエストの入力。
type RepeaterRequest struct {
	Method  string `json:"method"`
	URL     string `json:"url"`
	Headers string `json:"headers"`
	Body    string `json:"body"`
}
