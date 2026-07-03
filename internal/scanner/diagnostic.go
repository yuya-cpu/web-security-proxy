package scanner

import (
	schecks "github.com/yuya-cpu/security-checks/checks"
	"github.com/yuya-cpu/security-checks/model"
)

//
type DiagnosticResult = schecks.PassiveReport

//
type CookieAnalysis = schecks.CookieInfo

//
type Finding = model.Finding

//
type RiskLevel = model.RiskLevel

//
type DiagnosticScanner struct{}

func NewDiagnosticScanner() *DiagnosticScanner {
	return &DiagnosticScanner{}
}

//
func (d *DiagnosticScanner) Analyze(responseHeaders string, requestURL string, statusCode int) DiagnosticResult {
	return schecks.AnalyzePassiveFromText(requestURL, responseHeaders, statusCode)
}
