package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID             int64  `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"-"`
}

type UserStore struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*UserStore, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &UserStore{db: db}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *UserStore) Close() error {
	return s.db.Close()
}

func (s *UserStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			hashed_password TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`)
	if err != nil {
		return fmt.Errorf("migrate users: %w", err)
	}
	return nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	return s.get(ctx, `SELECT id, email, hashed_password FROM users WHERE email = ?`, email)
}

func (s *UserStore) GetByID(ctx context.Context, id int64) (User, error) {
	return s.get(ctx, `SELECT id, email, hashed_password FROM users WHERE id = ?`, id)
}

func (s *UserStore) Create(ctx context.Context, email, hashedPassword string) (User, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO users(email, hashed_password) VALUES(?, ?)`, email, hashedPassword)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("read created user id: %w", err)
	}
	return User{ID: id, Email: email, HashedPassword: hashedPassword}, nil
}

func (s *UserStore) get(ctx context.Context, query string, arg any) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, query, arg).Scan(&user.ID, &user.Email, &user.HashedPassword)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}
