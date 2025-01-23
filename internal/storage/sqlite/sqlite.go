package sqlite

import (
	"database/sql"
	"fmt"
	"url-shortener/internal/storage"

	"errors"

	"github.com/mattn/go-sqlite3"
)

// Storage represents a SQLite storage for URL shortening.
type Storage struct {
	db *sql.DB
}

func New(StoragePath string) (*Storage, error) {
	const op = "storage.sqlite.New"
	db, err := sql.Open("sqlite3", StoragePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`CREATE TABLE IF NOT EXISTS urls (
							id INTEGER PRIMARY KEY, 
							alias TEXT UNIQUE NOT NULL, 
							url TEXT NOT NULL);
							CREATE INDEX IF NOT EXISTS idx_alias ON urls (alias);`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"
	stmt, err := s.db.Prepare("INSERT INTO urls (alias, url) VALUES (?,?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: Alias already exists %w", op, storage.ErrURLExists)
		}
		return 0, fmt.Errorf("%s:  %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: Failed to get lastid %w", op, err)
	}
	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"
	row := s.db.QueryRow("SELECT url FROM urls WHERE alias=?", alias)

	var url string
	err := row.Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	} else if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"
	stmt, err := s.db.Prepare("DELETE FROM urls WHERE alias=?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: Failed to delete url: %w", op, err)
	}
	return nil
}
