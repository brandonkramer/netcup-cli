package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

type Manager struct {
	cfg   *config.Config
	oauth *oauth2.Config
	mu    sync.Mutex
	token *oauth2.Token
	creds *Credentials
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg: cfg,
		oauth: &oauth2.Config{
			ClientID: config.ClientID,
			Endpoint: oauth2.Endpoint{
				AuthURL:       config.OIDCIssuer + "/protocol/openid-connect/auth",
				TokenURL:      cfg.TokenURL(),
				DeviceAuthURL: cfg.DeviceAuthURL(),
			},
			Scopes: []string{"openid", "offline_access"},
		},
	}
}

func (m *Manager) Load() error {
	creds, err := LoadCredentials(m.cfg.CredentialsPath())
	if err != nil {
		return err
	}
	m.creds = creds
	m.token = &oauth2.Token{RefreshToken: creds.RefreshToken}
	return nil
}

func (m *Manager) HasCredentials() bool {
	_, err := os.Stat(m.cfg.CredentialsPath())
	return err == nil
}

func (m *Manager) Credentials() *Credentials {
	return m.creds
}

// DeviceLogin holds an in-flight device-code authorization.
type DeviceLogin struct {
	VerifyURL string
	UserCode  string
	resp      *oauth2.DeviceAuthResponse
	mgr       *Manager
}

// StartDeviceLogin begins OIDC device authorization. Call Wait to finish.
func (m *Manager) StartDeviceLogin(ctx context.Context) (*DeviceLogin, error) {
	resp, err := m.oauth.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("device auth: %w", err)
	}
	verifyURL := resp.VerificationURIComplete
	if verifyURL == "" {
		verifyURL = resp.VerificationURI
	}
	return &DeviceLogin{
		VerifyURL: verifyURL,
		UserCode:  resp.UserCode,
		resp:      resp,
		mgr:       m,
	}, nil
}

// Wait polls until the user completes device authorization.
func (d *DeviceLogin) Wait(ctx context.Context, save bool) (*Credentials, error) {
	tok, err := d.mgr.oauth.DeviceAccessToken(ctx, d.resp)
	if err != nil {
		return nil, fmt.Errorf("device token: %w", err)
	}
	if tok.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token returned; ensure offline_access scope is granted")
	}
	userID, username := claimsFromAccess(tok.AccessToken)
	creds := &Credentials{
		RefreshToken: tok.RefreshToken,
		UserID:       userID,
		Username:     username,
	}
	d.mgr.mu.Lock()
	d.mgr.token = tok
	d.mgr.creds = creds
	d.mgr.mu.Unlock()
	if save {
		if err := SaveCredentials(d.mgr.cfg.CredentialsPath(), creds); err != nil {
			return nil, err
		}
	}
	return creds, nil
}

func (m *Manager) Login(ctx context.Context, openBrowser, save bool) (*Credentials, error) {
	dev, err := m.StartDeviceLogin(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "Open this URL to authorize:\n\n  %s\n\nUser code: %s\n", dev.VerifyURL, dev.UserCode)
	if openBrowser {
		_ = openURL(dev.VerifyURL)
	}
	return dev.Wait(ctx, save)
}

func (m *Manager) Logout(revoke bool) error {
	if revoke {
		// Keycloak revoke is best-effort; account page is documented for app revoke.
		_ = revoke
	}
	m.mu.Lock()
	m.token = nil
	m.creds = nil
	m.mu.Unlock()
	return DeleteCredentials(m.cfg.CredentialsPath())
}

func (m *Manager) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if err := m.ensureLoaded(); err != nil {
		return nil, err
	}
	base := &oauth2.Token{RefreshToken: m.creds.RefreshToken}
	if m.token != nil {
		base = m.token
	}
	ts := m.oauth.TokenSource(ctx, base)
	return oauth2.ReuseTokenSource(nil, &persistingSource{
		mgr: m,
		src: ts,
	}), nil
}

type persistingSource struct {
	mgr *Manager
	src oauth2.TokenSource
}

func (p *persistingSource) Token() (*oauth2.Token, error) {
	tok, err := p.src.Token()
	if err != nil {
		return nil, err
	}
	p.mgr.mu.Lock()
	defer p.mgr.mu.Unlock()
	p.mgr.token = tok
	if tok.RefreshToken != "" && p.mgr.creds != nil && tok.RefreshToken != p.mgr.creds.RefreshToken {
		p.mgr.creds.RefreshToken = tok.RefreshToken
		_ = SaveCredentials(p.mgr.cfg.CredentialsPath(), p.mgr.creds)
	}
	if p.mgr.creds != nil && p.mgr.creds.UserID == "" {
		uid, user := claimsFromAccess(tok.AccessToken)
		p.mgr.creds.UserID = uid
		p.mgr.creds.Username = user
		_ = SaveCredentials(p.mgr.cfg.CredentialsPath(), p.mgr.creds)
	}
	return tok, nil
}

func (m *Manager) Refresh(ctx context.Context) (*oauth2.Token, error) {
	ts, err := m.TokenSource(ctx)
	if err != nil {
		return nil, err
	}
	return ts.Token()
}

func (m *Manager) UserID(ctx context.Context) (string, error) {
	if err := m.ensureLoaded(); err != nil {
		return "", err
	}
	if m.creds.UserID != "" {
		return m.creds.UserID, nil
	}
	tok, err := m.Refresh(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := claimsFromAccess(tok.AccessToken)
	if uid == "" {
		return "", fmt.Errorf("could not resolve user id from token")
	}
	m.creds.UserID = uid
	_ = SaveCredentials(m.cfg.CredentialsPath(), m.creds)
	return uid, nil
}

func (m *Manager) UserIDInt(ctx context.Context) (int32, error) {
	uid, err := m.UserID(ctx)
	if err != nil {
		return 0, err
	}
	n, err := ParseSCPUserID(uid)
	if err != nil {
		return 0, err
	}
	// Normalize stored credentials when they still hold a Keycloak-style sub.
	if m.creds != nil && m.creds.UserID != strconv.FormatInt(int64(n), 10) {
		m.creds.UserID = strconv.FormatInt(int64(n), 10)
		_ = SaveCredentials(m.cfg.CredentialsPath(), m.creds)
	}
	return n, nil
}

// ParseSCPUserID extracts the numeric SCP userId from a JWT sub or raw id.
// Keycloak subjects look like "f:<uuid>:<userId>"; the trailing segment is the SCP id.
func ParseSCPUserID(uid string) (int32, error) {
	if uid == "" {
		return 0, fmt.Errorf("empty user id")
	}
	if n, err := strconv.ParseInt(uid, 10, 32); err == nil {
		return int32(n), nil
	}
	// f:uuid:12345 → 12345
	if i := lastColon(uid); i >= 0 && i+1 < len(uid) {
		if n, err := strconv.ParseInt(uid[i+1:], 10, 32); err == nil {
			return int32(n), nil
		}
	}
	return 0, fmt.Errorf("invalid user id %q: expected numeric id or f:<uuid>:<id>", uid)
}

func lastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

func (m *Manager) Status(ctx context.Context) (map[string]any, error) {
	out := map[string]any{
		"logged_in": false,
		"profile":   m.cfg.Profile,
		"path":      m.cfg.CredentialsPath(),
	}
	if !m.HasCredentials() {
		return out, nil
	}
	if err := m.ensureLoaded(); err != nil {
		return out, err
	}
	out["logged_in"] = true
	out["user_id"] = m.creds.UserID
	out["username"] = m.creds.Username
	out["refresh_updated_at"] = m.creds.UpdatedAt
	out["refresh_age"] = time.Since(m.creds.UpdatedAt).Round(time.Second).String()

	tok, err := m.Refresh(ctx)
	if err != nil {
		out["access_token_error"] = err.Error()
		return out, nil
	}
	if !tok.Expiry.IsZero() {
		out["access_expires_at"] = tok.Expiry.UTC()
		out["access_expires_in"] = time.Until(tok.Expiry).Round(time.Second).String()
	}
	if m.creds.UserID == "" {
		uid, user := claimsFromAccess(tok.AccessToken)
		out["user_id"] = uid
		out["username"] = user
	}
	return out, nil
}

func (m *Manager) ensureLoaded() error {
	if m.creds != nil {
		return nil
	}
	if err := m.Load(); err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}
	return nil
}

func claimsFromAccess(access string) (userID, username string) {
	if access == "" {
		return "", ""
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	tok, _, err := parser.ParseUnverified(access, jwt.MapClaims{})
	if err != nil {
		return "", ""
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", ""
	}
	if sub, ok := claims["sub"].(string); ok {
		if n, err := ParseSCPUserID(sub); err == nil {
			userID = strconv.FormatInt(int64(n), 10)
		} else {
			userID = sub
		}
	}
	if pref, ok := claims["preferred_username"].(string); ok {
		username = pref
	}
	return userID, username
}

// OpenURL opens a URL in the default browser (best-effort).
func OpenURL(url string) error {
	return openURL(url)
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported OS for browser open")
	}
	return cmd.Start()
}
