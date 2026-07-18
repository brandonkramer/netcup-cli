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
	Tags      []string        `json:"tags,omitempty"`
}

type Store struct {
	dir     string
	enabled bool
	mu      sync.Mutex
}

func New(dir string, enabled bool) (*Store, error) {
	if enabled {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		_ = os.Chmod(dir, 0o700)
	}
	return &Store{dir: dir, enabled: enabled}, nil
}

func (s *Store) Enabled() bool { return s.enabled }

func (s *Store) SetEnabled(v bool) { s.enabled = v }

func Key(method, path, query, userID string) string {
	h := sha256.Sum256([]byte(strings.ToUpper(method) + "\n" + path + "\n" + query + "\n" + userID))
	return hex.EncodeToString(h[:16])
}

// ServersListKey matches the CLI `servers list` cache key (default filters).
func ServersListKey(userID string, q, name, ip string, sortBy []string, limit, offset, firewallPolicyID int32) string {
	return Key("GET", "/api/v1/servers",
		fmt.Sprintf("%s|%s|%s|%v|%d|%d|%d", q, name, ip, sortBy, limit, offset, firewallPolicyID),
		userID)
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

// Put stores body under key. Optional tags enable BustTags invalidation.
func (s *Store) Put(key string, body any, ttl time.Duration, tags ...string) error {
	if !s.enabled {
		return nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	e := Entry{
		FetchedAt: time.Now().UTC(),
		TTL:       ttl,
		Body:      raw,
		Tags:      uniqueTags(tags),
	}
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

func (s *Store) Delete(key string) error {
	if !s.enabled {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path(key))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
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
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) && name != prefix+".json" && !strings.Contains(name, prefix) {
			continue
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

// BustTags deletes every entry that was Put with any of the given tags.
func (s *Store) BustTags(tags ...string) {
	if !s.enabled || len(tags) == 0 {
		return
	}
	want := map[string]struct{}{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			want[t] = struct{}{}
		}
	}
	if len(want) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, de := range entries {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		path := filepath.Join(s.dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			continue
		}
		for _, t := range e.Tags {
			if _, ok := want[t]; ok {
				_ = os.Remove(path)
				break
			}
		}
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
	n := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") && !strings.HasSuffix(e.Name(), ".tmp") {
			n++
		}
	}
	return map[string]any{
		"entries": n,
		"dir":     s.dir,
		"enabled": s.enabled,
	}, nil
}

func (s *Store) path(key string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s.json", key))
}

func uniqueTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
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

const TagServers = "servers"
