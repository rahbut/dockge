// Package models provides application-level model logic on top of the raw DB
// query layer.
package models

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/rahbut/dockge/backend/db"
)

// ─── Settings KV store ────────────────────────────────────────────────────────

const settingCacheTTL = 60 * time.Second

type cacheEntry struct {
	value    string
	cachedAt time.Time
}

var (
	settingCache sync.Map // map[string]*cacheEntry
)

// GetSetting returns the string value for a key from the setting table.
// Results are cached for 60 seconds. Returns "" if the key does not exist.
func GetSetting(ctx context.Context, key string) (string, error) {
	// Check cache first.
	if v, ok := settingCache.Load(key); ok {
		entry := v.(*cacheEntry)
		if time.Since(entry.cachedAt) < settingCacheTTL {
			return entry.value, nil
		}
	}

	s, err := db.GetSettingByKey(ctx, key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	val := ""
	if s.Value.Valid {
		val = s.Value.String
	}

	settingCache.Store(key, &cacheEntry{value: val, cachedAt: time.Now()})
	return val, nil
}

// SetSetting persists a key/value pair and invalidates the cache entry.
func SetSetting(ctx context.Context, key, value, settingType string) error {
	if err := db.UpsertSetting(ctx, key, value, settingType); err != nil {
		return err
	}
	settingCache.Delete(key)
	return nil
}

// DeleteSetting removes a setting and invalidates the cache.
func DeleteSetting(ctx context.Context, key string) error {
	if err := db.DeleteSetting(ctx, key); err != nil {
		return err
	}
	settingCache.Delete(key)
	return nil
}

// GetSettingsByType returns all settings of the given type as a map[key]value.
func GetSettingsByType(ctx context.Context, settingType string) (map[string]string, error) {
	rows, err := db.GetSettingsByType(ctx, settingType)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Value.Valid {
			out[r.Key] = r.Value.String
		} else {
			out[r.Key] = ""
		}
	}
	return out, nil
}

// ─── Typed helpers ────────────────────────────────────────────────────────────

// GetDisableAuth returns the disableAuth setting (true if value == "1" or "true").
func GetDisableAuth(ctx context.Context) (bool, error) {
	v, err := GetSetting(ctx, "disableAuth")
	if err != nil {
		return false, err
	}
	return v == "1" || v == "true", nil
}

// GetJWTSecret returns the jwtSecret setting value.
func GetJWTSecret(ctx context.Context) (string, error) {
	return GetSetting(ctx, "jwtSecret")
}

// GetTrustProxy returns the trustProxy setting.
func GetTrustProxy(ctx context.Context) (bool, error) {
	v, err := GetSetting(ctx, "trustProxy")
	if err != nil {
		return false, err
	}
	return v == "1" || v == "true", nil
}

// GetPrimaryHostname returns the primaryHostname setting.
func GetPrimaryHostname(ctx context.Context) (string, error) {
	return GetSetting(ctx, "primaryHostname")
}

// GetUpdateCheckTime returns the scheduled update check time as "HH:MM",
// or "" if the feature is disabled.
func GetUpdateCheckTime(ctx context.Context) (string, error) {
	return GetSetting(ctx, "updateCheckTime")
}
