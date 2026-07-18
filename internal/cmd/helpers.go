package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/wait"
)

func fmtAny(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func fmtBool(v any) string {
	b, ok := v.(bool)
	if !ok {
		return fmtAny(v)
	}
	if b {
		return "true"
	}
	return "false"
}

func fmtPtr[T any](v *T) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(*v)
}

func serverProjection(s scpclient.Server) map[string]any {
	state := ""
	if s.ServerLiveInfo != nil && s.ServerLiveInfo.State != nil {
		state = string(*s.ServerLiveInfo.State)
	}
	ipv4 := []string{}
	if s.Ipv4Addresses != nil {
		for _, ip := range *s.Ipv4Addresses {
			if ip.Ip != nil {
				ipv4 = append(ipv4, *ip.Ip)
			}
		}
	}
	ipv6 := []string{}
	if s.Ipv6Addresses != nil {
		for _, ip := range *s.Ipv6Addresses {
			if ip.NetworkPrefix != nil {
				ipv6 = append(ipv6, *ip.NetworkPrefix)
			}
		}
	}
	site := ""
	if s.Site != nil {
		site = s.Site.City
	}
	m := map[string]any{
		"id":                 fmtPtr(s.Id),
		"name":               fmtPtr(s.Name),
		"nickname":           fmtPtr(s.Nickname),
		"hostname":           fmtPtr(s.Hostname),
		"state":              state,
		"ipv4":               strings.Join(ipv4, ","),
		"ipv6":               strings.Join(ipv6, ","),
		"site":               site,
		"rescueSystemActive": s.RescueSystemActive != nil && *s.RescueSystemActive,
	}
	if app.Flags.Full {
		m["full"] = s
	}
	return m
}

func serverListProjection(s scpclient.ServerListMinimal) map[string]any {
	return map[string]any{
		"id":       fmtPtr(s.Id),
		"name":     fmtPtr(s.Name),
		"nickname": fmtPtr(s.Nickname),
		"hostname": fmtPtr(s.Hostname),
		"state":    "",
		"ipv4":     "",
		"site":     "",
	}
}

func serversTable(list []scpclient.ServerListMinimal) output.TableData {
	rows := make([][]string, 0, len(list))
	raw := make([]map[string]any, 0, len(list))
	for _, s := range list {
		p := serverListProjection(s)
		raw = append(raw, p)
		rows = append(rows, []string{
			fmtAny(p["id"]),
			fmtAny(p["name"]),
			fmtAny(p["nickname"]),
			fmtAny(p["hostname"]),
			fmtBool(s.Disabled != nil && *s.Disabled),
		})
	}
	return output.TableData{
		Headers: []string{"id", "name", "nickname", "hostname", "disabled"},
		Rows:    rows,
		Raw:     raw,
	}
}

func decodeServersList(resp *scpclient.GetApiV1ServersResponse) ([]scpclient.ServerListMinimal, error) {
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	if resp.HALJSON200 != nil {
		return *resp.HALJSON200, nil
	}
	var list []scpclient.ServerListMinimal
	if err := json.Unmarshal(resp.Body, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func decodeServer(resp *scpclient.GetApiV1ServersServerIdResponse) (*scpclient.Server, error) {
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.HALJSON200 != nil {
		return resp.HALJSON200, nil
	}
	var s scpclient.Server
	if err := json.Unmarshal(resp.Body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func successData(format output.Format, table output.TableData) any {
	if format == output.FormatTable {
		return table
	}
	if format == output.FormatBrief || format == output.FormatJSONL {
		if raw, ok := table.Raw.([]map[string]any); ok {
			out := make([]any, len(raw))
			for i, r := range raw {
				out[i] = r
			}
			return out
		}
	}
	if raw, ok := table.Raw.([]map[string]any); ok {
		return raw
	}
	if table.Raw != nil {
		return table.Raw
	}
	return table
}

func emitTaskResult(command string, httpStatus int, task *scpclient.TaskInfo, data any) error {
	app.Out.Command = command
	opts := []func(*output.Envelope){output.WithHTTPStatus(httpStatus)}
	if task != nil {
		opts = append(opts, output.WithTask(wait.TaskEnvelope(task)))
	}
	return app.Out.Success(data, opts...)
}

func parseInt32(s string) (int32, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, output.Exit(output.ExitUsage, fmt.Sprintf("invalid integer: %s", s))
	}
	return int32(n), nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func firstTask(tasks ...*scpclient.TaskInfo) *scpclient.TaskInfo {
	for _, t := range tasks {
		if t != nil {
			return t
		}
	}
	return nil
}

func taskFromBody(status int, body []byte, candidates ...*scpclient.TaskInfo) *scpclient.TaskInfo {
	if t := firstTask(candidates...); t != nil {
		return t
	}
	if status == 202 && len(body) > 0 {
		var t scpclient.TaskInfo
		if err := json.Unmarshal(body, &t); err == nil && t.Uuid != nil {
			return &t
		}
	}
	return nil
}

func firstJSON(a, b any, body []byte) any {
	if a != nil {
		return a
	}
	if b != nil {
		return b
	}
	if len(body) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body)
	}
	return v
}
