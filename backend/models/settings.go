// Package models provides application-level model logic on top of the raw DB
// query layer.
package models

import (
	"context"
	"database/sql"
	"encoding/json"
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

// ─── Update check result cache ────────────────────────────────────────────────

const lastUpdateResultsKey = "lastUpdateResults"

// UpdateCheckEntry holds the persisted result for a single stack.
type UpdateCheckEntry struct {
	UpdateAvailable bool                       `json:"updateAvailable"`
	Services        map[string]ServiceSnapshot `json:"services"`
}

// ServiceSnapshot is a minimal, serialisable record of a per-service result.
type ServiceSnapshot struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	Error           string `json:"error,omitempty"`
}

// SetLastUpdateResults marshals allResults (keyed by stack name) and persists
// the JSON blob in the setting table so late-connecting clients can see the
// last-known update status after a scheduled check.
//
// allResults values are expected to be map[string]any with keys
// "updateAvailable" (bool) and "services" (map[string]ServiceUpdateResult-like).
func SetLastUpdateResults(ctx context.Context, allResults map[string]any) error {
	// Convert the generic map to a typed, JSON-safe structure.
	typed := make(map[string]UpdateCheckEntry, len(allResults))
	for name, v := range allResults {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		entry := UpdateCheckEntry{}
		if av, ok := m["updateAvailable"].(bool); ok {
			entry.UpdateAvailable = av
		}
		if svcs, ok := m["services"].(map[string]any); ok {
			entry.Services = make(map[string]ServiceSnapshot, len(svcs))
			for svcName, sv := range svcs {
				if sm, ok := sv.(map[string]any); ok {
					snap := ServiceSnapshot{}
					if a, ok := sm["updateAvailable"].(bool); ok {
						snap.UpdateAvailable = a
					}
					if e, ok := sm["error"].(string); ok {
						snap.Error = e
					}
					entry.Services[svcName] = snap
				}
			}
		}
		typed[name] = entry
	}

	raw, err := json.Marshal(typed)
	if err != nil {
		return err
	}
	return SetSetting(ctx, lastUpdateResultsKey, string(raw), "internal")
}

// GetLastUpdateResults returns the last-persisted update check results, keyed
// by stack name. Returns an empty map (never nil) when no results have been
// stored yet.
func GetLastUpdateResults(ctx context.Context) (map[string]UpdateCheckEntry, error) {
	raw, err := GetSetting(ctx, lastUpdateResultsKey)
	if err != nil || raw == "" {
		return map[string]UpdateCheckEntry{}, err
	}
	var out map[string]UpdateCheckEntry
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]UpdateCheckEntry{}, nil
	}
	return out, nil
}
