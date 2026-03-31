package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Stack struct {
	ID           int        `json:"Id"`
	Name         string     `json:"Name"`
	Status       int        `json:"Status"`
	EndpointID   int        `json:"EndpointId"`
	CreationDate int64      `json:"CreationDate"`
	UpdateDate   int64      `json:"UpdateDate"`
	GitConfig    *GitConfig `json:"GitConfig"`
}

type GitConfig struct {
	URL            string `json:"URL"`
	ReferenceName  string `json:"ReferenceName"`
	ConfigHash     string `json:"ConfigHash"`
	ConfigFilePath string `json:"ConfigFilePath"`
}

func (s Stack) StatusLabel() string {
	if s.Status == 1 {
		return "active"
	}
	return "inactive"
}

func (c *Client) ListStacks() ([]Stack, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/stacks", nil)
	if err != nil {
		return nil, err
	}

	var stacks []Stack
	if err := c.do(req, &stacks); err != nil {
		return nil, fmt.Errorf("failed to list stacks: %w", err)
	}
	return stacks, nil
}

func (c *Client) FindStackByName(name string) (*Stack, error) {
	stacks, err := c.ListStacks()
	if err != nil {
		return nil, err
	}
	for _, s := range stacks {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, nil
}

type CreateStackPayload struct {
	Name                      string          `json:"name"`
	RepositoryURL             string          `json:"repositoryURL"`
	RepositoryReferenceName   string          `json:"repositoryReferenceName"`
	ComposeFile               string          `json:"composeFile"`
	RepositoryAuthentication  bool            `json:"repositoryAuthentication"`
	RepositoryGitCredentialID int             `json:"repositoryGitCredentialID,omitempty"`
	AutoUpdate                *AutoUpdate     `json:"autoUpdate,omitempty"`
	Env                       []EnvVar        `json:"env,omitempty"`
	AdditionalFiles           []string        `json:"additionalFiles,omitempty"`
	SupportRelativePath       bool            `json:"supportRelativePath"`
	FilesystemPath            string          `json:"filesystemPath,omitempty"`
}

type AutoUpdate struct {
	Webhook string `json:"webhook"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *Client) CreateStack(endpointID int, payload CreateStackPayload) (*Stack, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/stacks/create/standalone/repository?endpointId=%d", c.BaseURL, endpointID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var stack Stack
	if err := c.do(req, &stack); err != nil {
		return nil, fmt.Errorf("failed to create stack: %w", err)
	}
	return &stack, nil
}

func (c *Client) DeleteStack(stackID, endpointID int) error {
	url := fmt.Sprintf("%s/api/stacks/%d?endpointId=%d", c.BaseURL, stackID, endpointID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to delete stack: %w", err)
	}
	return nil
}

type StackDetail struct {
	ID         int        `json:"Id"`
	Name       string     `json:"Name"`
	Status     int        `json:"Status"`
	EndpointID int        `json:"EndpointId"`
	Env        []EnvVar   `json:"Env"`
	GitConfig  *GitConfig `json:"GitConfig"`
}

func (c *Client) GetStack(stackID int) (*StackDetail, error) {
	url := fmt.Sprintf("%s/api/stacks/%d", c.BaseURL, stackID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var stack StackDetail
	if err := c.do(req, &stack); err != nil {
		return nil, fmt.Errorf("failed to get stack: %w", err)
	}
	return &stack, nil
}

type UpdateStackPayload struct {
	Env                       []EnvVar    `json:"env"`
	RepositoryReferenceName   string      `json:"repositoryReferenceName"`
	RepositoryAuthentication  bool        `json:"repositoryAuthentication"`
	RepositoryGitCredentialID int         `json:"repositoryGitCredentialID,omitempty"`
	Prune                     bool        `json:"prune"`
	PullImage                 bool        `json:"pullImage"`
}

func (c *Client) UpdateStackEnv(stackID, endpointID int, payload UpdateStackPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use the git-specific update endpoint for git-backed stacks
	url := fmt.Sprintf("%s/api/stacks/%d/git?endpointId=%d", c.BaseURL, stackID, endpointID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to update stack env: %w", err)
	}
	return nil
}

type RedeployPayload struct {
	RepositoryReferenceName   string `json:"repositoryReferenceName"`
	RepositoryAuthentication  bool   `json:"repositoryAuthentication"`
	RepositoryGitCredentialID int    `json:"repositoryGitCredentialID,omitempty"`
	PullImage                 bool   `json:"pullImage"`
}

func (c *Client) RedeployStack(stackID, endpointID int, payload RedeployPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/stacks/%d/git/redeploy?endpointId=%d", c.BaseURL, stackID, endpointID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to redeploy stack: %w", err)
	}
	return nil
}
