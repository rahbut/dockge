// Package db handles SQLite initialisation, pragma configuration,
// and provides all typed query functions for the application.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // CGO-free SQLite driver
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB is the global database handle, set by Init.
var DB *sql.DB

// Init opens (or creates) the SQLite database at dbPath, applies WAL-mode
// pragmas, and runs any outstanding goose migrations.
func Init(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("db: create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("db: open: %w", err)
	}

	// SQLite works best with a single writer connection.
	db.SetMaxOpenConns(1)

	// Apply pragmas matching the original Node.js configuration.
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA cache_size = -12000",
		"PRAGMA auto_vacuum = INCREMENTAL",
		"PRAGMA synchronous = NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("db: pragma %q: %w", p, err)
		}
	}

	DB = db

	// Run migrations via goose (embedded SQL files).
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.Up(DB, "migrations"); err != nil {
		return fmt.Errorf("db: migrations: %w", err)
	}

	return nil
}

// Close closes the database.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// ─── Setting queries ──────────────────────────────────────────────────────────

// Setting represents one row in the setting table.
type Setting struct {
	ID    int64
	Key   string
	Value sql.NullString
	Type  sql.NullString
}

// GetSettingByKey returns the setting for the given key, or sql.ErrNoRows.
func GetSettingByKey(ctx context.Context, key string) (*Setting, error) {
	row := DB.QueryRowContext(ctx,
		`SELECT id, key, value, type FROM setting WHERE key = ? LIMIT 1`, key)
	s := &Setting{}
	err := row.Scan(&s.ID, &s.Key, &s.Value, &s.Type)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// UpsertSetting inserts or updates a setting by key.
func UpsertSetting(ctx context.Context, key, value, settingType string) error {
	_, err := DB.ExecContext(ctx,
		`INSERT INTO setting (key, value, type) VALUES (?, ?, ?)
         ON CONFLICT(key) DO UPDATE SET value = excluded.value, type = excluded.type`,
		key, value, settingType)
	return err
}

// GetSettingsByType returns all settings whose type column matches the given value.
func GetSettingsByType(ctx context.Context, settingType string) ([]Setting, error) {
	rows, err := DB.QueryContext(ctx,
		`SELECT id, key, value, type FROM setting WHERE type = ?`, settingType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.Type); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteSetting removes a setting by key.
func DeleteSetting(ctx context.Context, key string) error {
	_, err := DB.ExecContext(ctx, `DELETE FROM setting WHERE key = ?`, key)
	return err
}

// ─── User queries ─────────────────────────────────────────────────────────────

// User represents one row in the user table.
type User struct {
	ID       int64
	Username string
	Password sql.NullString
	Active   bool
}

// GetUserByUsername returns an active user by username, or sql.ErrNoRows.
func GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := DB.QueryRowContext(ctx,
		`SELECT id, username, password, active FROM user WHERE username = ? AND active = 1 LIMIT 1`,
		username)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.Active)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// CountUsers returns the total number of user rows.
func CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := DB.QueryRowContext(ctx, `SELECT COUNT(id) FROM user`).Scan(&count)
	return count, err
}

// InsertUser creates a new user and returns the new row ID.
func InsertUser(ctx context.Context, username, passwordHash string) (int64, error) {
	res, err := DB.ExecContext(ctx,
		`INSERT INTO user (username, password, active) VALUES (?, ?, 1)`,
		username, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateUserPassword sets a new bcrypt hash for the user with the given ID.
func UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	_, err := DB.ExecContext(ctx,
		`UPDATE user SET password = ? WHERE id = ?`, passwordHash, userID)
	return err
}

// ─── Agent queries ────────────────────────────────────────────────────────────

// Agent represents one row in the agent table.
type Agent struct {
	ID       int64
	URL      string
	Username string
	Password string
	Active   bool
}

// GetAllAgents returns all agent rows.
func GetAllAgents(ctx context.Context) ([]Agent, error) {
	rows, err := DB.QueryContext(ctx,
		`SELECT id, url, username, password, active FROM agent`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.URL, &a.Username, &a.Password, &a.Active); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// InsertAgent persists a new agent and returns the new row ID.
func InsertAgent(ctx context.Context, url, username, password string) (int64, error) {
	res, err := DB.ExecContext(ctx,
		`INSERT INTO agent (url, username, password, active) VALUES (?, ?, ?, 1)`,
		url, username, password)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteAgentByURL removes an agent by its URL. Returns the deleted agent's
// row, or sql.ErrNoRows if not found.
func DeleteAgentByURL(ctx context.Context, url string) (*Agent, error) {
	a, err := GetAgentByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	_, err = DB.ExecContext(ctx, `DELETE FROM agent WHERE url = ?`, url)
	return a, err
}

// GetAgentByURL returns an agent by URL, or sql.ErrNoRows.
func GetAgentByURL(ctx context.Context, url string) (*Agent, error) {
	row := DB.QueryRowContext(ctx,
		`SELECT id, url, username, password, active FROM agent WHERE url = ? LIMIT 1`, url)
	a := &Agent{}
	err := row.Scan(&a.ID, &a.URL, &a.Username, &a.Password, &a.Active)
	if err != nil {
		return nil, err
	}
	return a, nil
}
