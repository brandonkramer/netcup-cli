package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPutGetTTL(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "c"), true)
	if err != nil {
		t.Fatal(err)
	}
	key := Key("GET", "/api/v1/servers", "", "1")
	if err := s.Put(key, []string{"a"}, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	body, _, _, ok := s.Get(key)
	if !ok || body == nil {
		t.Fatal("expected cache hit")
	}
	time.Sleep(60 * time.Millisecond)
	if _, _, _, ok := s.Get(key); ok {
		t.Fatal("expected TTL expiry")
	}
}
