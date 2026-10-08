package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"io"

	"github.com/Aeomar999/CommPit/core"
)

func (s *Store) Put(ctx context.Context, id string, r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	err = s.execContext(ctx,
		`INSERT INTO blobs (id, data) VALUES (?, ?)`, id, data)
	return int64(len(data)), err
}

func (s *Store) Get(ctx context.Context, id string) (io.ReadCloser, error) {
	row := s.queryRowContext(ctx, `SELECT data FROM blobs WHERE id = ?`, id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewInternal("blob not found")
		}
		return nil, err
	}
	return &nopReadCloser{data: data}, nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	err := s.execContext(ctx, `DELETE FROM blobs WHERE id = ?`, id)
	return err
}

type nopReadCloser struct {
	data []byte
	pos  int
}

func (n *nopReadCloser) Read(p []byte) (int, error) {
	if n.pos >= len(n.data) {
		return 0, io.EOF
	}
	nCopied := copy(p, n.data[n.pos:])
	n.pos += nCopied
	return nCopied, nil
}

func (n *nopReadCloser) Close() error { return nil }
