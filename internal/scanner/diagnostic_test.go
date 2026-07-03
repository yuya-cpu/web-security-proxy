package scanner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yuya-cpu/security-checks/model"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
)

func TestDiagnosticScanner_Analyze(t *testing.T) {
	s := scanner.NewDiagnosticScanner()
	result := s.Analyze(`Server: nginx/1.18.0
Content-Type: text/html
Set-Cookie: sid=abc; Path=/`, "https://example.com", 200)

	assert.Equal(t, "nginx/1.18.0", result.Server)
	assert.NotEmpty(t, result.Findings)
	assert.Len(t, result.Cookies, 1)
}

func TestDiagnosticScanner_DetectsMissingSecurityHeaders(t *testing.T) {
	s := scanner.NewDiagnosticScanner()
	result := s.Analyze("X-Frame-Options: DENY", "https://example.com", 200)

	hasWarning := false
	for _, finding := range result.Findings {
		if finding.RiskLevel.IsWarning() {
			hasWarning = true
			break
		}
	}
	assert.True(t, hasWarning)
}

func TestDiagnosticScanner_ServerNotDisclosed(t *testing.T) {
	s := scanner.NewDiagnosticScanner()
	result := s.Analyze("Content-Type: text/html", "http://example.com", 200)
	assert.Equal(t, "(not disclosed)", result.Server)
}

func TestRiskLevelIsWarning(t *testing.T) {
	assert.False(t, model.RiskPass.IsWarning())
	assert.True(t, model.RiskHigh.IsWarning())
}
