package tui

import (
	"strings"
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

func TestServerItemFilterValue(t *testing.T) {
	id := int32(42)
	name := "vps1"
	nick := "web"
	host := "vps1.example"
	it := serverItem{server: scpclient.ServerListMinimal{
		Id: &id, Name: &name, Nickname: &nick, Hostname: &host,
	}}
	got := it.FilterValue()
	for _, want := range []string{"vps1", "web", "vps1.example", "42"} {
		if !strings.Contains(got, want) {
			t.Fatalf("FilterValue %q missing %q", got, want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	sec := int32(90061) // 1d 1h 1m
	if got := formatUptime(&sec); got != "1d 1h 1m" {
		t.Fatalf("got %q", got)
	}
}
