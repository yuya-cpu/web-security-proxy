package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
)

//
type ScanService struct {
	scanner *scanner.ActiveScanner
}

func NewScanService(scanner *scanner.ActiveScanner) *ScanService {
	return &ScanService{scanner: scanner}
}

func (s *ScanService) ScanTransaction(ctx context.Context, tx *model.HTTPTransaction) (scanner.ActiveScanReport, error) {
	if tx == nil {
		return scanner.ActiveScanReport{}, fmt.Errorf("transaction is nil")
	}

	url := strings.TrimSpace(tx.URL)
	if url == "" {
		return scanner.ActiveScanReport{}, fmt.Errorf("url is required")
	}
	if strings.EqualFold(tx.Method, "CONNECT") {
		return scanner.ActiveScanReport{}, fmt.Errorf("CONNECT transactions cannot be scanned")
	}

	return s.scanner.Scan(ctx, url, tx.ResponseHeaders, tx.StatusCode)
}
