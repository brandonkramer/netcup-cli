package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	FetchedAt time.Time       `json:"fetched_at"`
	TTL       time.Duration   `json:"ttl"`
	Body      json.RawMessage `json:"body"`
	ETag      string          `json:"etag,omitempty"`
}

type Store struct {
	dir     string
	enabled bool
	mu      sync.Mutex
}

func New(dir string, enabled bool) (*Store, error) {
	if enabled {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &Store{dir: dir, enabled: enabled}, nil
}

func (s *Store) Enabled() bool { return s.enabled }

func (s *Store) SetEnabled(v bool) { s.enabled = v }

func Key(method, path, query, userID string) string {
	h := sha256.Sum256([]byte(strings.ToUpper(method) + "\n" + path + "\n" + query + "\n" + userID))
	return hex.EncodeToString(h[:16])
}

func (s *Store) Get(key string) (json.RawMessage, time.Duration, time.Duration, bool) {
	if !s.enabled {
		return nil, 0, 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.path(key)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, false
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, 0, 0, false
	}
	age := time.Since(e.FetchedAt)
	if age > e.TTL {
		_ = os.Remove(path)
		return nil, 0, 0, false
	}
	return e.Body, age, e.TTL, true
}

func (s *Store) Put(key string, body any, ttl time.Duration) error {
	if !s.enabled {
		return nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	e := Entry{FetchedAt: time.Now().UTC(), TTL: ttl, Body: raw}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tmp := s.path(key) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(key))
}

func (s *Store) DeletePrefix(prefix string) (int, error) {
	if !s.enabled {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		name := e.Name()
		if prefix != "" && !strings.HasPrefix(name, prefix) && name != prefix+".json" {
			// also allow tag-based bust via sidecar prefixes in key name
			if !strings.Contains(name, prefix) {
				continue
			}
		}
		if err := os.Remove(filepath.Join(s.dir, name)); err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Store) Clear() (int, error) {
	return s.DeletePrefix("")
}

func (s *Store) BustTags(tags ...string) {
	if !s.enabled {
		return
	}
	for _, tag := range tags {
		_, _ = s.DeletePrefix(tag)
	}
}

func (s *Store) Stats() (map[string]any, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"entries": 0, "dir": s.dir, "enabled": s.enabled}, nil
		}
		return nil, err
	}
	return map[string]any{
		"entries": len(entries),
		"dir":     s.dir,
		"enabled": s.enabled,
	}, nil
}

func (s *Store) path(key string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s.json", key))
}

// TTL defaults from SPEC §5.2
const (
	TTLServerList   = 60 * time.Second
	TTLServerGet    = 15 * time.Second
	TTLRelatedList  = 30 * time.Second
	TTLUserLists    = 120 * time.Second
	TTLCatalog      = 10 * time.Minute
	TTLAuthProfile  = 5 * time.Minute
	TTLOpenAPI      = 24 * time.Hour
	TTLMaintenance  = 30 * time.Second
	TTLTaskTerminal = 1 * time.Hour
)
