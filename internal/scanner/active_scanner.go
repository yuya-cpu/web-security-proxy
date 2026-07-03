package scanner

import (
	"context"

	schecks "github.com/yuya-cpu/security-checks/checks"
	"github.com/yuya-cpu/security-checks/fetch"
)

//
type ActiveScanReport = schecks.ActiveScanReport

//
type ActiveScanner struct {
	fetcher fetch.Fetcher
}

func NewActiveScanner(fetcher fetch.Fetcher) *ActiveScanner {
	return &ActiveScanner{fetcher: fetcher}
}

func (s *ActiveScanner) Scan(ctx context.Context, targetURL, responseHeaders string, statusCode int) (ActiveScanReport, error) {
	baseline := schecks.BuildResponse(targetURL, statusCode, responseHeaders)
	return schecks.RunActiveScan(ctx, s.fetcher, targetURL, baseline)
}
