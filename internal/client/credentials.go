package client

import (
	"fmt"
	"net/http"
)

type User struct {
	ID       int    `json:"Id"`
	Username string `json:"Username"`
}

type GitCredential struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (c *Client) GetCurrentUser() (*User, error) {
	// Try /api/users/me first
	req, err := http.NewRequest("GET", c.BaseURL+"/api/users/me", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := c.do(req, &user); err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return &user, nil
}

func (c *Client) ListGitCredentials(userID int) ([]GitCredential, error) {
	// Portainer 2.45 moved them to shared credentials; older versions keep them per user
	var lastErr error
	for _, path := range []string{"/api/cloud/gitcredentials", fmt.Sprintf("/api/users/%d/gitcredentials", userID)} {
		req, err := http.NewRequest("GET", c.BaseURL+path, nil)
		if err != nil {
			return nil, err
		}
		var creds []GitCredential
		if err := c.do(req, &creds); err != nil {
			lastErr = err
			continue
		}
		return creds, nil
	}
	return nil, fmt.Errorf("failed to list git credentials: %w", lastErr)
}
