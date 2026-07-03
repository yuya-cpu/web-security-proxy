package service

import (
	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
)

//
type DiagnosticService struct {
	scanner *scanner.DiagnosticScanner
}

func NewDiagnosticService(scanner *scanner.DiagnosticScanner) *DiagnosticService {
	return &DiagnosticService{scanner: scanner}
}

func (s *DiagnosticService) AnalyzeTransaction(tx *model.HTTPTransaction) scanner.DiagnosticResult {
	if tx == nil {
		return scanner.DiagnosticResult{Server: "(not disclosed)", OverallRisk: "PASS"}
	}
	return s.scanner.Analyze(tx.ResponseHeaders, tx.URL, tx.StatusCode)
}
