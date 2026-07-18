package tui

import (
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

func TestFirewallPresetBody(t *testing.T) {
	ssh, err := firewallPresetBody("policy-preset-ssh")
	if err != nil {
		t.Fatal(err)
	}
	if ssh.Name != "allow-ssh" || ssh.Rules == nil || len(*ssh.Rules) != 1 {
		t.Fatalf("ssh preset: %+v", ssh)
	}
	if (*ssh.Rules)[0].DestinationPorts == nil || *(*ssh.Rules)[0].DestinationPorts != "22" {
		t.Fatalf("ssh port: %+v", (*ssh.Rules)[0])
	}
	if (*ssh.Rules)[0].Protocol != scpclient.TCP || (*ssh.Rules)[0].Direction != scpclient.INGRESS {
		t.Fatalf("ssh rule meta: %+v", (*ssh.Rules)[0])
	}

	http, err := firewallPresetBody("policy-preset-http")
	if err != nil {
		t.Fatal(err)
	}
	if http.Name != "allow-http" || http.Rules == nil || len(*http.Rules) != 2 {
		t.Fatalf("http preset: %+v", http)
	}
}
