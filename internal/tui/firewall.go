package tui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Firewall loads server interfaces first, which is the authoritative MAC source,
// then appends the user's policy library to the same searchable list.
func loadFirewallCmd(ctx context.Context, client *scpclient.ClientWithResponses, serverID, userID int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetApiV1ServersServerIdInterfacesWithResponse(ctx, serverID, nil)
		if err != nil {
			return resourceLoadedMsg{tab: "firewall", err: err}
		}
		if resp == nil {
			return resourceLoadedMsg{tab: "firewall", err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "firewall", err: apiStatusErr("interfaces", resp.StatusCode(), resp.Body)}
		}
		items := resourceItems("mac", firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))
		pol, err := client.GetApiV1UsersUserIdFirewallPoliciesWithResponse(ctx, userID, nil)
		if err != nil {
			return resourceLoadedMsg{tab: "firewall", items: items, warning: "policies: " + err.Error()}
		}
		if pol == nil {
			return resourceLoadedMsg{tab: "firewall", items: items, warning: "policies: empty response"}
		}
		if pol.StatusCode() == 401 {
			return resourceLoadedMsg{tab: "firewall", err: apiStatusErr("policies", pol.StatusCode(), pol.Body)}
		}
		if pol.StatusCode() == 200 {
			items = append(items, resourceItems("policy", firstJSON(pol.JSON200, pol.HALJSON200, pol.Body))...)
		} else {
			return resourceLoadedMsg{tab: "firewall", items: items, warning: fmt.Sprintf("policies HTTP %d", pol.StatusCode())}
		}
		return resourceLoadedMsg{tab: "firewall", items: items}
	}
}

func (m model) updateFirewallKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.firewallList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	it, _ := m.firewallList.SelectedItem().(resourceItem)
	switch msg.String() {
	case "R":
		m.loading = true
		return m, loadFirewallCmd(m.ctx, m.deps.Client, m.server.ID, m.deps.UserID)
	case "enter":
		if it.kind == "policy" {
			return m, loadPolicyCmd(m.ctx, m.deps.Client, m.deps.UserID, it.id)
		}
		if it.id != "" {
			return m, loadFirewallDetailCmd(m.ctx, m.deps.Client, m.server.ID, it.id)
		}
	case "r":
		if it.id != "" {
			m.confirm = &confirmState{action: "firewall-reapply", names: []string{it.title}, extra: it.id}
			return m, nil
		}
	case "c":
		if it.id != "" {
			m.confirm = &confirmState{action: "firewall-restore", names: []string{it.title}, extra: it.id}
			return m, nil
		}
	case "s":
		if it.id != "" && it.kind != "policy" {
			return m.beginResourceEditCtx("firewall-set", it.id, "@file or inline JSON")
		}
	case "p":
		m.resourceMode = "policies"
		m.status = "policy library loaded"
		return m, nil
	case "n":
		return m.beginResourceEdit("policy-create", "@file or inline JSON")
	case "S":
		m.confirm = &confirmState{
			action: "policy-preset-ssh",
			names:  []string{"create policy allow-ssh (TCP 22 ingress)"},
			extra:  "Creates a user policy only — does not attach/activate on a NIC. Use s to assign after.",
		}
		return m, nil
	case "H":
		m.confirm = &confirmState{
			action: "policy-preset-http",
			names:  []string{"create policy allow-http (TCP 80+443 ingress)"},
			extra:  "Creates a user policy only — does not attach/activate on a NIC. Use s to assign after.",
		}
		return m, nil
	case "e":
		if it.kind == "policy" {
			return m.beginResourceEditCtx("policy-update", it.id, "@file or inline JSON")
		}
	case "d":
		if it.kind == "policy" {
			m.confirm = &confirmState{action: "policy-delete", names: []string{it.title}, extra: it.id}
			return m, nil
		}
	}
	return m.updateActiveList(msg)
}

func loadFirewallDetailCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, mac string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetApiV1ServersServerIdInterfacesMacFirewallWithResponse(ctx, id, mac, nil)
		if err != nil {
			return resourceLoadedMsg{tab: "firewall", err: err}
		}
		if resp == nil {
			return resourceLoadedMsg{tab: "firewall", err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "firewall", err: apiStatusErr("firewall", resp.StatusCode(), resp.Body)}
		}
		return resourceLoadedMsg{tab: "firewall", detail: prettyJSON(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))}
	}
}
func loadPolicyCmd(ctx context.Context, client *scpclient.ClientWithResponses, uid int32, policy string) tea.Cmd {
	return func() tea.Msg {
		n := parseID(policy)
		resp, err := client.GetApiV1UsersUserIdFirewallPoliciesIdWithResponse(ctx, uid, n)
		if err != nil {
			return resourceLoadedMsg{tab: "firewall", err: err}
		}
		if resp == nil {
			return resourceLoadedMsg{tab: "firewall", err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "firewall", err: apiStatusErr("policy", resp.StatusCode(), resp.Body)}
		}
		return resourceLoadedMsg{tab: "firewall", detail: prettyJSON(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))}
	}
}

func (m model) firewallView() string {
	right := stylePanelTitle.Render("Firewall") + "\n\n" + styleMuted.Render("MACs from NICs. enter get · r reapply · c restore · s assign JSON · S/H create policy presets (not applied) · n policy")
	if m.resourceDetail != "" {
		right = m.viewport.View()
	}
	return m.twoPanels(m.firewallList.View(), right)
}

func firewallPresetCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, preset string) tea.Cmd {
	return func() tea.Msg {
		body, err := firewallPresetBody(preset)
		if err != nil {
			return resourceActionMsg{action: preset, err: err}
		}
		r, e := c.PostApiV1UsersUserIdFirewallPoliciesWithResponse(ctx, uid, body)
		if e != nil {
			return resourceActionMsg{action: preset, err: e}
		}
		if r == nil {
			return resourceActionMsg{action: preset, err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 201 && status != 202 {
			return resourceActionMsg{action: preset, status: status, err: apiStatusErr(preset, status, r.Body)}
		}
		return resourceActionMsg{action: preset, status: status, task: extractTask(status, r.Body)}
	}
}

func firewallPresetBody(preset string) (scpclient.FirewallPolicySave, error) {
	desc := func(s string) *string { return &s }
	port := func(s string) *string { return &s }
	switch preset {
	case "policy-preset-ssh":
		rules := []scpclient.FirewallRule{{
			Action:           scpclient.ACCEPT,
			Description:      desc("SSH"),
			Direction:        scpclient.INGRESS,
			Protocol:         scpclient.TCP,
			DestinationPorts: port("22"),
		}}
		d := "Allow inbound SSH (TCP 22)"
		return scpclient.FirewallPolicySave{Name: "allow-ssh", Description: &d, Rules: &rules}, nil
	case "policy-preset-http":
		rules := []scpclient.FirewallRule{
			{
				Action:           scpclient.ACCEPT,
				Description:      desc("HTTP"),
				Direction:        scpclient.INGRESS,
				Protocol:         scpclient.TCP,
				DestinationPorts: port("80"),
			},
			{
				Action:           scpclient.ACCEPT,
				Description:      desc("HTTPS"),
				Direction:        scpclient.INGRESS,
				Protocol:         scpclient.TCP,
				DestinationPorts: port("443"),
			},
		}
		d := "Allow inbound HTTP/HTTPS (TCP 80, 443)"
		return scpclient.FirewallPolicySave{Name: "allow-http", Description: &d, Rules: &rules}, nil
	default:
		return scpclient.FirewallPolicySave{}, tuiError("unknown preset")
	}
}

func bodyArg(value string) ([]byte, error) {
	if strings.HasPrefix(value, "@") {
		return os.ReadFile(strings.TrimPrefix(value, "@"))
	}
	return []byte(value), nil
}
func firewallSetCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, mac, value string) tea.Cmd {
	return func() tea.Msg {
		body, err := bodyArg(value)
		if err != nil {
			return resourceActionMsg{action: "firewall-set", err: err}
		}
		resp, err := client.PutApiV1ServersServerIdInterfacesMacFirewallWithBodyWithResponse(ctx, id, mac, "application/json", bytes.NewReader(body))
		if err != nil {
			return resourceActionMsg{action: "firewall-set", err: err}
		}
		if resp == nil {
			return resourceActionMsg{action: "firewall-set", err: tuiError("empty response")}
		}
		status := resp.StatusCode()
		if status != 200 && status != 202 {
			return resourceActionMsg{action: "firewall-set", status: status, err: apiStatusErr("firewall set", status, resp.Body)}
		}
		return resourceActionMsg{action: "firewall-set", status: status, task: extractTask(status, resp.Body, resp.JSON202, resp.HALJSON202)}
	}
}
