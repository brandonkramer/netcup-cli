package cmd

import (
	"net/http"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check SCP availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "ping"
			url := app.Cfg.APIRoot() + "/api/ping"
			client := &http.Client{Timeout: 10 * time.Second}
			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", "*/*")
			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				return app.Out.Fail(output.ExitAPI, "api", "", err.Error(), 0, nil)
			}
			defer resp.Body.Close()
			ok := resp.StatusCode >= 200 && resp.StatusCode < 300
			data := map[string]any{
				"ok":      ok,
				"status":  resp.StatusCode,
				"took_ms": time.Since(start).Milliseconds(),
				"url":     url,
			}
			if !ok {
				return app.Out.Fail(output.ExitAPI, "api", "", "ping failed", resp.StatusCode, nil)
			}
			if app.Out.Format == output.FormatTable {
				return app.Out.Success(output.TableData{
					Headers: []string{"ok", "status", "took_ms"},
					Rows:    [][]string{{"true", fmtAny(resp.StatusCode), fmtAny(data["took_ms"])}},
					Raw:     data,
				}, output.WithHTTPStatus(resp.StatusCode))
			}
			return app.Out.Success(data, output.WithHTTPStatus(resp.StatusCode))
		},
	}
}

func newMaintenanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "maintenance",
		Short: "Show SCP maintenance information",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "maintenance"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1MaintenanceWithResponse(cmd.Context())
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("maintenance", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}
