package selectserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

// Resolve turns id|name|nickname|ip into a server id.
func Resolve(ctx context.Context, client *scpclient.ClientWithResponses, selector string) (int32, *scpclient.Server, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return 0, nil, output.Exit(output.ExitUsage, "server selector required (use --server/-s)")
	}
	if id, err := strconv.ParseInt(selector, 10, 32); err == nil {
		srv, err := getServer(ctx, client, int32(id))
		if err != nil {
			return 0, nil, err
		}
		return int32(id), srv, nil
	}

	list, err := listServers(ctx, client, &scpclient.GetApiV1ServersParams{Q: &selector})
	if err != nil {
		return 0, nil, err
	}
	matches := make([]scpclient.ServerListMinimal, 0)
	for _, s := range list {
		if matchMinimal(s, selector) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		list2, err := listServers(ctx, client, &scpclient.GetApiV1ServersParams{Ip: &selector})
		if err == nil {
			matches = append(matches, list2...)
		}
	}
	if len(matches) == 0 {
		return 0, nil, output.Exit(output.ExitNotFound, fmt.Sprintf("no server matching %q", selector))
	}
	if len(matches) > 1 {
		return 0, nil, output.Exit(output.ExitUsage, fmt.Sprintf("ambiguous server selector %q (%d matches); use id", selector, len(matches)))
	}
	s := matches[0]
	if s.Id == nil {
		return 0, nil, fmt.Errorf("server missing id")
	}
	srv, err := getServer(ctx, client, *s.Id)
	if err != nil {
		return *s.Id, nil, err
	}
	return *s.Id, srv, nil
}

func listServers(ctx context.Context, client *scpclient.ClientWithResponses, params *scpclient.GetApiV1ServersParams) ([]scpclient.ServerListMinimal, error) {
	resp, err := client.GetApiV1ServersWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("list servers: HTTP %d", resp.StatusCode())
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	if resp.HALJSON200 != nil {
		return *resp.HALJSON200, nil
	}
	var list []scpclient.ServerListMinimal
	if err := json.Unmarshal(resp.Body, &list); err != nil {
		return nil, fmt.Errorf("list servers: decode: %w", err)
	}
	return list, nil
}

func getServer(ctx context.Context, client *scpclient.ClientWithResponses, id int32) (*scpclient.Server, error) {
	resp, err := client.GetApiV1ServersServerIdWithResponse(ctx, id, &scpclient.GetApiV1ServersServerIdParams{
		LoadServerLiveInfo: ptr(true),
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 404 {
		return nil, output.Exit(output.ExitNotFound, fmt.Sprintf("server %d not found", id))
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("get server: HTTP %d", resp.StatusCode())
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.HALJSON200 != nil {
		return resp.HALJSON200, nil
	}
	var s scpclient.Server
	if err := json.Unmarshal(resp.Body, &s); err != nil {
		return nil, fmt.Errorf("get server: decode: %w", err)
	}
	return &s, nil
}

func matchMinimal(s scpclient.ServerListMinimal, selector string) bool {
	sel := strings.ToLower(selector)
	if s.Name != nil && strings.EqualFold(*s.Name, selector) {
		return true
	}
	if s.Nickname != nil && strings.EqualFold(*s.Nickname, selector) {
		return true
	}
	if s.Hostname != nil && strings.EqualFold(*s.Hostname, selector) {
		return true
	}
	if s.Name != nil && strings.Contains(strings.ToLower(*s.Name), sel) {
		return true
	}
	if s.Nickname != nil && strings.Contains(strings.ToLower(*s.Nickname), sel) {
		return true
	}
	return false
}

func ptr[T any](v T) *T { return &v }
