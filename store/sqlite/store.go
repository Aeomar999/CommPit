package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/Aeomar999/CommPit/core"
	"github.com/oklog/ulid/v2"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type Store struct {
	writeDB *sql.DB
	readDB  *sql.DB
	mu      sync.Mutex
	closed  bool
}

func NewStore(dataDir string, readPoolSize int) (*Store, error) {
	if readPoolSize <= 0 {
		readPoolSize = 4
	}

	var dsn string
	if dataDir == ":memory:" {
		dsn = fmt.Sprintf("file:mocksms-%s?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", ulid.Make().String())
	} else {
		// Use filepath.ToSlash to ensure forward slashes for SQLite URI
		dataDir = filepath.ToSlash(dataDir)
		dsn = fmt.Sprintf("file:%s/mocksms.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)", dataDir)
	}

	writeDB, err := openDB(dsn, 1)
	if err != nil {
		return nil, fmt.Errorf("open write db: %w", err)
	}

	readDB, err := openDB(dsn, readPoolSize)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("open read db: %w", err)
	}

	if err := runMigrations(writeDB); err != nil {
		writeDB.Close()
		readDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &Store{
		writeDB: writeDB,
		readDB:  readDB,
	}, nil
}

func openDB(dsn string, maxConns int) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

//go:embed migrations/*.sql
var migrations embed.FS

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

func (s *Store) getReadDB() *sql.DB {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.readDB == nil {
		return s.writeDB
	}
	return s.readDB
}

func (s *Store) getWriteDB() *sql.DB {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeDB
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true

	var errs []error
	if err := s.writeDB.Close(); err != nil {
		errs = append(errs, err)
	}
	if s.readDB != nil {
		if err := s.readDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close: %v", errs)
	}
	return nil
}

func (s *Store) execContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := s.getWriteDB().ExecContext(ctx, query, args...)
	return err
}

func (s *Store) execContextWithResult(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.getWriteDB().ExecContext(ctx, query, args...)
}

func (s *Store) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.getReadDB().QueryContext(ctx, query, args...)
}

func (s *Store) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.getReadDB().QueryRowContext(ctx, query, args...)
}

func (s *Store) Transaction(ctx context.Context, fn func(core.Store) error) error {
	tx, err := s.getWriteDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txStore := &txStore{tx: tx, base: s}
	err = fn(txStore)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

type txStore struct {
	tx   *sql.Tx
	base *Store
}

func (t *txStore) Transaction(ctx context.Context, fn func(core.Store) error) error {
	// Nested transactions not supported, just execute the function
	return fn(t)
}
