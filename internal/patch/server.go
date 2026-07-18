package patch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

const mergePatch = "application/merge-patch+json"

func Server(
	ctx context.Context,
	client *scpclient.ClientWithResponses,
	serverID int32,
	body any,
	stateOption *string,
) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	params := &scpclient.PatchApiV1ServersServerIdParams{StateOption: stateOption}
	return client.PatchApiV1ServersServerIdWithBodyWithResponse(ctx, serverID, params, mergePatch, bytes.NewReader(raw))
}

func SetState(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, state scpclient.ServerState1, stateOption *string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, scpclient.ServerStatePatch{State: &state}, stateOption)
}

func SetHostname(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, hostname string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, scpclient.ServerHostnamePatch{Hostname: &hostname}, nil)
}

func SetNickname(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, nickname string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, map[string]any{"nickname": nickname}, nil)
}

func SetUEFI(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, uefi bool) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, scpclient.ServerUEFIPatch{Uefi: &uefi}, nil)
}

func SetBootorder(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, order []scpclient.Bootorder) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, map[string]any{"bootorder": order}, nil)
}

func SetRootPassword(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, password string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, map[string]any{"rootPassword": password}, nil)
}

func SetAutostart(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, autostart bool) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, scpclient.ServerAutostartPatch{Autostart: &autostart}, nil)
}

func SetOsOptimization(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, opt scpclient.OsOptimization) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, scpclient.ServerOsOptimizationPatch{OsOptimization: &opt}, nil)
}

func SetCpuTopology(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, sockets, coresPerSocket int32) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	topo := scpclient.CpuTopology{SocketCount: &sockets, CoresPerSocketCount: &coresPerSocket}
	return Server(ctx, client, serverID, scpclient.ServerCpuTopologyPatch{CpuTopology: &topo}, nil)
}

func SetKeyboardLayout(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, layout string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
	return Server(ctx, client, serverID, scpclient.ServerKeyboardLayoutPatch{KeyboardLayout: &layout}, nil)
}

func RequireOK(resp *scpclient.PatchApiV1ServersServerIdResponse) (*scpclient.TaskInfo, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil response")
	}
	switch resp.StatusCode() {
	case 202:
		if resp.HALJSON202 != nil {
			return resp.HALJSON202, nil
		}
		if resp.JSON202 != nil {
			return resp.JSON202, nil
		}
		var t scpclient.TaskInfo
		if err := json.Unmarshal(resp.Body, &t); err == nil && t.Uuid != nil {
			return &t, nil
		}
		return nil, fmt.Errorf("202 without task body")
	case 200:
		return nil, nil
	default:
		return nil, fmt.Errorf("patch server: HTTP %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
