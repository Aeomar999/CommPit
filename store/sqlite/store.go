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
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type Store struct {
	writeDB *sql.DB
	readDBs []*sql.DB
	mu      sync.Mutex
	closed  bool
}

func NewStore(dataDir string, readPoolSize int) (*Store, error) {
	if readPoolSize <= 0 {
		readPoolSize = 4
	}

	writeDB, err := openDB(dataDir, true)
	if err != nil {
		return nil, fmt.Errorf("open write db: %w", err)
	}

	readDBs := make([]*sql.DB, readPoolSize)
	for i := 0; i < readPoolSize; i++ {
		readDB, err := openDB(dataDir, false)
		if err != nil {
			writeDB.Close()
			for j := 0; j < i; j++ {
				readDBs[j].Close()
			}
			return nil, fmt.Errorf("open read db %d: %w", i, err)
		}
		readDBs[i] = readDB
	}

	if err := runMigrations(writeDB); err != nil {
		writeDB.Close()
		for _, db := range readDBs {
			db.Close()
		}
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &Store{
		writeDB: writeDB,
		readDBs: readDBs,
	}, nil
}

func openDB(dataDir string, write bool) (*sql.DB, error) {
	var dsn string
	if dataDir == ":memory:" {
		dsn = "file::memory:?cache=shared"
	} else {
		// Use filepath.ToSlash to ensure forward slashes for SQLite URI
		dataDir = filepath.ToSlash(dataDir)
		dsn = fmt.Sprintf("file:%s/mocksms.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", dataDir)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if write {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}

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
	if s.closed || len(s.readDBs) == 0 {
		return s.writeDB
	}
	return s.readDBs[0]
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
	for _, db := range s.readDBs {
		if err := db.Close(); err != nil {
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

func (t *txStore) execContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.ExecContext(ctx, query, args...)
	return err
}

func (t *txStore) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *txStore) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}
