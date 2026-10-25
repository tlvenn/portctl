package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// A Portainer 2.45 git source: the repo's URL and, through a shared credential, its authentication.
type gitSource struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Interval   string `json:"interval"`
	Provider   string `json:"provider"`
	Connection struct {
		TLSSkipVerify  bool `json:"tlsSkipVerify"`
		Authentication *struct {
			Type               string `json:"type"`
			SharedCredentialID int    `json:"sharedCredentialId"`
		} `json:"authentication"`
	} `json:"connection"`
}

// ResaveGitSource saves a git source again, unchanged. Portainer 2.45 then copies the source's shared
// credential to every stack that uses it: a stack created from the source has none of its own until
// then, and its redeploys fail with "Unable get latest commit id".
func (c *Client) ResaveGitSource(id int) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/gitops/sources/%d", c.BaseURL, id), nil)
	if err != nil {
		return err
	}
	var src gitSource
	if err := c.do(req, &src); err != nil {
		return fmt.Errorf("failed to read git source %d: %w", id, err)
	}
	auth := src.Connection.Authentication
	if auth == nil || auth.SharedCredentialID == 0 {
		return nil // no shared credential to hand on
	}

	payload := map[string]any{
		"name":          src.Name,
		"url":           src.URL,
		"interval":      src.Interval,
		"tlsSkipVerify": src.Connection.TLSSkipVerify,
		"authentication": map[string]any{
			"sharedCredentialId": auth.SharedCredentialID,
			"type":               auth.Type,
			"provider":           src.Provider,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err = http.NewRequest("PUT", fmt.Sprintf("%s/api/gitops/sources/%d", c.BaseURL, id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to save git source %d: %w", id, err)
	}
	return nil
}

// EnsureStackGitCredential saves the git source until the stack shows its credential. Portainer
// can clear the stack's copy shortly after a create or an update returns, so one save may come too
// early.
func (c *Client) EnsureStackGitCredential(stackID, sourceID int) error {
	for attempt := 0; attempt < 5; attempt++ {
		time.Sleep(2 * time.Second)
		if err := c.ResaveGitSource(sourceID); err != nil {
			return err
		}
		time.Sleep(time.Second)
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/stacks/%d", c.BaseURL, stackID), nil)
		if err != nil {
			return err
		}
		var st struct {
			GitConfig *struct {
				Authentication *struct {
					GitCredentialID int `json:"GitCredentialID"`
				} `json:"Authentication"`
			} `json:"GitConfig"`
		}
		if err := c.do(req, &st); err != nil {
			return err
		}
		if st.GitConfig != nil && st.GitConfig.Authentication != nil && st.GitConfig.Authentication.GitCredentialID != 0 {
			return nil
		}
	}
	return fmt.Errorf("stack %d still has no git credential after saving source %d: redeploys will fail until it does", stackID, sourceID)
}
