package cmd

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/auth"
	"github.com/brandonkramer/netcup-cli/internal/cache"
	"github.com/brandonkramer/netcup-cli/internal/config"
	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/wait"
	"golang.org/x/oauth2"
)

type GlobalFlags struct {
	Format       string
	Yes          bool
	NoWait       bool
	WaitTimeout  time.Duration
	PollInterval time.Duration
	Cache        *bool
	NoCache      bool
	CacheTTL     time.Duration
	Profile      string
	UserID       string
	Server       string
	Quiet        bool
	Verbose      bool
	ConfigDir    string
	Full         bool
}

type App struct {
	Flags      GlobalFlags
	Cfg        *config.Config
	Auth       *auth.Manager
	Cache      *cache.Store
	Out        *output.Printer
	Client     *scpclient.ClientWithResponses
	HTTPClient *http.Client
}

func (a *App) Init() error {
	cfg, err := config.Load(a.Flags.Profile, a.Flags.ConfigDir)
	if err != nil {
		return err
	}
	if a.Flags.Profile != "" {
		if err := config.ValidateProfile(a.Flags.Profile); err != nil {
			return err
		}
		cfg.Profile = a.Flags.Profile
	}
	a.Cfg = cfg

	format := a.Flags.Format
	if format == "" {
		format = cfg.Format
	}
	if format != "" {
		if _, err := output.ParseFormat(format); err != nil {
			return output.Exit(output.ExitUsage, err.Error())
		}
	}
	a.Out = output.NewPrinter(format)
	a.Out.Quiet = a.Flags.Quiet
	a.Out.Verbose = a.Flags.Verbose

	enabled := cfg.CacheEnabled
	if a.Flags.NoCache {
		enabled = false
	}
	if a.Flags.Cache != nil {
		enabled = *a.Flags.Cache
	}
	store, err := cache.New(cfg.CacheDir(), enabled)
	if err != nil {
		return err
	}
	a.Cache = store
	a.Auth = auth.NewManager(cfg)
	return nil
}

func (a *App) EnsureClient(ctx context.Context) error {
	if a.Client != nil {
		return nil
	}
	if !a.Auth.HasCredentials() {
		return output.Exit(output.ExitAuth, "not logged in; run: netcup auth login")
	}
	ts, err := a.Auth.TokenSource(ctx)
	if err != nil {
		return output.Exit(output.ExitAuth, err.Error())
	}
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &oauth2.Transport{
			Source: ts,
			Base:   http.DefaultTransport,
		},
	}
	client, err := scpclient.NewClientWithResponses(a.Cfg.APIRoot(), scpclient.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}
	a.HTTPClient = httpClient
	a.Client = client
	return nil
}

func (a *App) ResolveUserID(ctx context.Context) (int32, error) {
	if a.Flags.UserID != "" {
		n, err := strconv.ParseInt(a.Flags.UserID, 10, 32)
		if err != nil || n < 1 || n > math.MaxInt32 {
			return 0, output.Exit(output.ExitUsage, "invalid --user-id (want positive integer)")
		}
		return int32(n), nil
	}
	return a.Auth.UserIDInt(ctx)
}

func (a *App) WaitOpts() wait.Options {
	to := a.Flags.WaitTimeout
	if to == 0 {
		to = a.Cfg.WaitTimeout
	}
	pi := a.Flags.PollInterval
	if pi == 0 {
		pi = a.Cfg.PollInterval
	}
	return wait.Options{
		Timeout:      to,
		PollInterval: pi,
		NoWait:       a.Flags.NoWait,
		OnProgress: func(task scpclient.TaskInfo) {
			if a.Out.Format == output.FormatJSON || a.Out.Format == output.FormatJSONL || a.Out.Quiet {
				return
			}
			state := ""
			if task.State != nil {
				state = string(*task.State)
			}
			a.Out.Info(fmt.Sprintf("task %s: %s (%.0f%%)", deref(task.Uuid), state, wait.ProgressPercent(task)))
		},
	}
}

func (a *App) Confirm(action string) error {
	if a.Flags.Yes {
		return nil
	}
	fi, err := os.Stdin.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return output.Exit(output.ExitUsage, fmt.Sprintf("confirmation required for %s; re-run with --yes", action))
	}
	fmt.Fprintf(os.Stderr, "Confirm %s? [y/N] ", action)
	var ans string
	_, _ = fmt.Scanln(&ans)
	if ans != "y" && ans != "Y" && ans != "yes" {
		return output.Exit(output.ExitUsage, "aborted")
	}
	return nil
}

func (a *App) HandleAPIError(command string, status int, body []byte) error {
	a.Out.Command = command
	msg := fmt.Sprintf("HTTP %d", status)
	if len(body) > 0 {
		msg = fmt.Sprintf("HTTP %d: %s", status, truncate(string(body), 300))
	}
	code := output.ExitAPI
	if status == 404 {
		code = output.ExitNotFound
	}
	if status == 401 || status == 403 {
		code = output.ExitAuth
	}
	return a.Out.Fail(code, "api", fmt.Sprintf("%d", status), msg, status, nil)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptr[T any](v T) *T { return &v }
