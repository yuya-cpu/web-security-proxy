package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
)

//
type TrafficRepository interface {
	Save(ctx context.Context, tx *model.HTTPTransaction) (int64, error)
	List(ctx context.Context, limit int) ([]model.HTTPTransaction, error)
	GetByID(ctx context.Context, id int64) (*model.HTTPTransaction, error)
}

//
type SQLiteTrafficRepository struct {
	db *sql.DB
}

//
func NewSQLiteTrafficRepository(dbPath string, migrationSQL string) (*SQLiteTrafficRepository, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec(migrationSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migration: %w", err)
	}

	return &SQLiteTrafficRepository{db: db}, nil
}

func (r *SQLiteTrafficRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteTrafficRepository) Save(ctx context.Context, tx *model.HTTPTransaction) (int64, error) {
	query := `
		INSERT INTO http_transactions (
			method, url, request_headers, request_body,
			status_code, response_headers, response_body, duration_ms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	createdAt := tx.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	result, err := r.db.ExecContext(ctx, query,
		tx.Method,
		tx.URL,
		tx.RequestHeaders,
		tx.RequestBody,
		tx.StatusCode,
		tx.ResponseHeaders,
		tx.ResponseBody,
		tx.DurationMS,
		createdAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	return id, nil
}

func (r *SQLiteTrafficRepository) List(ctx context.Context, limit int) ([]model.HTTPTransaction, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, method, url, request_headers, request_body,
		       status_code, response_headers, response_body, duration_ms, created_at
		FROM http_transactions
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var items []model.HTTPTransaction
	for rows.Next() {
		item, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return items, nil
}

func (r *SQLiteTrafficRepository) GetByID(ctx context.Context, id int64) (*model.HTTPTransaction, error) {
	query := `
		SELECT id, method, url, request_headers, request_body,
		       status_code, response_headers, response_body, duration_ms, created_at
		FROM http_transactions
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	item, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return item, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTransaction(row scannable) (*model.HTTPTransaction, error) {
	var item model.HTTPTransaction
	var createdAt string

	err := row.Scan(
		&item.ID,
		&item.Method,
		&item.URL,
		&item.RequestHeaders,
		&item.RequestBody,
		&item.StatusCode,
		&item.ResponseHeaders,
		&item.ResponseBody,
		&item.DurationMS,
		&createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan transaction: %w", err)
	}

	parsed, err := time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
	}
	item.CreatedAt = parsed.UTC()

	return &item, nil
}
