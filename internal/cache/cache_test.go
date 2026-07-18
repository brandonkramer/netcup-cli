package cache

import (
	"fmt"
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

func TestBustTags(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "c"), true)
	if err != nil {
		t.Fatal(err)
	}
	k1 := ServersListKey("1", "", "", "", nil, 0, 0, 0)
	k2 := Key("GET", "/api/v1/other", "", "1")
	if err := s.Put(k1, []string{"servers"}, time.Hour, TagServers); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(k2, []string{"other"}, time.Hour); err != nil {
		t.Fatal(err)
	}
	s.BustTags(TagServers)
	if _, _, _, ok := s.Get(k1); ok {
		t.Fatal("expected servers entry busted")
	}
	if _, _, _, ok := s.Get(k2); !ok {
		t.Fatal("expected untagged entry to remain")
	}
}

func TestServersListKeyMatchesCLI(t *testing.T) {
	uid := "12345"
	var sortBy []string
	q, name, ip := "", "", ""
	var limit, offset, firewallPolicyID int32
	legacy := Key("GET", "/api/v1/servers",
		fmt.Sprintf("%s|%s|%s|%v|%d|%d|%d", q, name, ip, sortBy, limit, offset, firewallPolicyID),
		uid)
	got := ServersListKey(uid, q, name, ip, sortBy, limit, offset, firewallPolicyID)
	if got != legacy {
		t.Fatalf("ServersListKey diverged from CLI format:\n got %s\nwant %s", got, legacy)
	}
}
