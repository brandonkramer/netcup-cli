package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Credentials struct {
	RefreshToken string    `json:"refresh_token"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserID       string    `json:"user_id,omitempty"`
	Username     string    `json:"username,omitempty"`
}

func LoadCredentials(path string) (*Credentials, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	if c.RefreshToken == "" {
		return nil, fmt.Errorf("credentials missing refresh_token")
	}
	return &c, nil
}

func SaveCredentials(path string, c *Credentials) error {
	c.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func DeleteCredentials(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
