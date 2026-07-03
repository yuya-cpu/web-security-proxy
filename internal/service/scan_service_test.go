package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

func TestScanService_Validation(t *testing.T) {
	svc := service.NewScanService(scanner.NewActiveScanner(nil))

	_, err := svc.ScanTransaction(context.Background(), nil)
	require.Error(t, err)

	_, err = svc.ScanTransaction(context.Background(), &model.HTTPTransaction{Method: "GET"})
	require.Error(t, err)

	_, err = svc.ScanTransaction(context.Background(), &model.HTTPTransaction{
		Method: "CONNECT",
		URL:    "https://example.com:443",
	})
	require.Error(t, err)
}
