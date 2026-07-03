package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

func TestDiagnosticService_AnalyzeTransaction(t *testing.T) {
	svc := service.NewDiagnosticService(scanner.NewDiagnosticScanner())

	result := svc.AnalyzeTransaction(&model.HTTPTransaction{
		URL:              "https://example.com",
		StatusCode:       200,
		ResponseHeaders:  "Server: test-server\nSet-Cookie: token=1; Path=/",
	})

	assert.Equal(t, "test-server", result.Server)
	require.Len(t, result.Cookies, 1)
	assert.NotEmpty(t, result.Findings)
}

func TestDiagnosticService_NilTransaction(t *testing.T) {
	svc := service.NewDiagnosticService(scanner.NewDiagnosticScanner())
	result := svc.AnalyzeTransaction(nil)
	assert.Equal(t, "(not disclosed)", result.Server)
}
