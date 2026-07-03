package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
)

const maxBodySize = 1 << 20 // 1MB

//
type TrafficRecorder interface {
	SaveTransaction(ctx context.Context, tx *model.HTTPTransaction) (int64, error)
}

//
type Server struct {
	recorder TrafficRecorder
	client   *http.Client
}

//
func NewServer(recorder TrafficRecorder) *Server {
	return &Server{
		recorder: recorder,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

//
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
		return
	}
	s.handleHTTP(w, r)
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	reqBody, err := readLimitedBody(r.Body, maxBodySize)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(reqBody))

	outReq := r.Clone(ctx)
	outReq.RequestURI = ""
	if outReq.URL.Scheme == "" {
		outReq.URL.Scheme = "http"
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = outReq.Host
	}

	resp, err := s.client.Do(outReq)
	if err != nil {
		s.saveErrorTransaction(ctx, r, reqBody, start, err)
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := readLimitedBody(resp.Body, maxBodySize)
	if err != nil {
		http.Error(w, "failed to read response body", http.StatusBadGateway)
		return
	}

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)

	s.saveTransaction(ctx, outReq, reqBody, resp, respBody, start)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		s.saveConnectTransaction(ctx, r, start, 500, fmt.Errorf("hijacking not supported"))
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	targetConn, err := net.DialTimeout("tcp", r.Host, 30*time.Second)
	if err != nil {
		s.saveConnectTransaction(ctx, r, start, 502, err)
		http.Error(w, "failed to connect", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		s.saveConnectTransaction(ctx, r, start, 500, err)
		http.Error(w, "failed to hijack", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	s.saveConnectTransaction(ctx, r, start, 200, nil)

	errCh := make(chan error, 2)
	go pipe(clientConn, targetConn, errCh)
	go pipe(targetConn, clientConn, errCh)

	<-errCh
}

func (s *Server) saveTransaction(ctx context.Context, req *http.Request, reqBody []byte, resp *http.Response, respBody []byte, start time.Time) {
	if s.recorder == nil {
		return
	}

	tx := &model.HTTPTransaction{
		Method:          req.Method,
		URL:             req.URL.String(),
		RequestHeaders:  formatHeaders(req.Header),
		RequestBody:     string(reqBody),
		StatusCode:      resp.StatusCode,
		ResponseHeaders: formatHeaders(resp.Header),
		ResponseBody:    string(respBody),
		DurationMS:      time.Since(start).Milliseconds(),
		CreatedAt:       time.Now().UTC(),
	}

	_, _ = s.recorder.SaveTransaction(ctx, tx)
}

func (s *Server) saveErrorTransaction(ctx context.Context, req *http.Request, reqBody []byte, start time.Time, proxyErr error) {
	if s.recorder == nil {
		return
	}

	tx := &model.HTTPTransaction{
		Method:         req.Method,
		URL:            requestURL(req),
		RequestHeaders: formatHeaders(req.Header),
		RequestBody:    string(reqBody),
		StatusCode:     http.StatusBadGateway,
		ResponseHeaders: "X-Proxy-Error: true",
		ResponseBody:   proxyErr.Error(),
		DurationMS:     time.Since(start).Milliseconds(),
		CreatedAt:      time.Now().UTC(),
	}

	_, _ = s.recorder.SaveTransaction(ctx, tx)
}

func (s *Server) saveConnectTransaction(ctx context.Context, req *http.Request, start time.Time, status int, connErr error) {
	if s.recorder == nil {
		return
	}

	body := "HTTPS tunnel established (content is encrypted in Phase 1)"
	if connErr != nil {
		body = connErr.Error()
	}

	tx := &model.HTTPTransaction{
		Method:         http.MethodConnect,
		URL:            "https://" + req.Host,
		RequestHeaders: formatHeaders(req.Header),
		RequestBody:    "",
		StatusCode:     status,
		ResponseHeaders: "Note: HTTPS body is not decrypted in Phase 1",
		ResponseBody:   body,
		DurationMS:     time.Since(start).Milliseconds(),
		CreatedAt:      time.Now().UTC(),
	}

	_, _ = s.recorder.SaveTransaction(ctx, tx)
}

func readLimitedBody(body io.ReadCloser, limit int64) ([]byte, error) {
	defer body.Close()
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

func formatHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := headers.Write(&buf); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func requestURL(req *http.Request) string {
	if req.URL != nil && req.URL.String() != "" {
		return req.URL.String()
	}
	return req.Host
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func pipe(dst net.Conn, src net.Conn, errCh chan<- error) {
	_, err := io.Copy(dst, src)
	if tcpConn, ok := dst.(*net.TCPConn); ok {
		_ = tcpConn.CloseWrite()
	}
	errCh <- err
}

//
func DumpRequest(req *http.Request) (string, error) {
	data, err := httputil.DumpRequest(req, true)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
