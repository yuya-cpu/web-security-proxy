package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/repository"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS http_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,
    url TEXT NOT NULL,
    request_headers TEXT NOT NULL DEFAULT '',
    request_body TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    response_headers TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func newTestRepository(t *testing.T) repository.TrafficRepository {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	repo, err := repository.NewSQLiteTrafficRepository(dbPath, migrationSQL)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close()
	})

	return repo
}

func TestSQLiteTrafficRepository_SaveAndGet(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	tx := &model.HTTPTransaction{
		Method:          "GET",
		URL:             "http://example.com/",
		RequestHeaders:  "Host: example.com",
		RequestBody:     "",
		StatusCode:      200,
		ResponseHeaders: "Content-Type: text/html",
		ResponseBody:    "<html></html>",
		DurationMS:      12,
		CreatedAt:       time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
	}

	id, err := repo.Save(ctx, tx)
	require.NoError(t, err)
	assert.Positive(t, id)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, "http://example.com/", got.URL)
	assert.Equal(t, 200, got.StatusCode)
}

func TestSQLiteTrafficRepository_List(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := repo.Save(ctx, &model.HTTPTransaction{
			Method:     "GET",
			URL:        "http://example.com/" + string(rune('a'+i)),
			StatusCode: 200,
		})
		require.NoError(t, err)
	}

	items, err := repo.List(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestSQLiteTrafficRepository_GetByID_NotFound(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.GetByID(context.Background(), 999)
	require.Error(t, err)
}

func TestNewSQLiteTrafficRepository_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "db", "proxy.db")

	repo, err := repository.NewSQLiteTrafficRepository(dbPath, migrationSQL)
	require.NoError(t, err)
	defer repo.Close()

	_, err = os.Stat(filepath.Dir(dbPath))
	require.NoError(t, err)
}
