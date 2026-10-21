package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	PortainerURL     string
	APIKey           string
	EndpointID       int
	EndpointOverride string // from --endpoint flag; name or numeric ID; wins over EndpointID
	GitRepo          string
	GitBranch        string
	GitCredentialID  int
	RepoPath         string
	FilesystemPath   string
	EnvFile          string // KEY=VALUE file applied to stack env on deploy and redeploy
}

func Load() (*Config, error) {
	var missing []string

	portainerURL := os.Getenv("PORTAINER_URL")
	if portainerURL == "" {
		missing = append(missing, "PORTAINER_URL")
	}

	apiKey := os.Getenv("PORTAINER_API_KEY")
	if apiKey == "" {
		missing = append(missing, "PORTAINER_API_KEY")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("required environment variables not set: %s", strings.Join(missing, ", "))
	}

	u, err := url.Parse(portainerURL)
	if err != nil {
		return nil, fmt.Errorf("PORTAINER_URL is not a valid URL: %w", err)
	}
	if u.Scheme == "http" {
		fmt.Fprintf(os.Stderr, "Warning: PORTAINER_URL uses http:// — API key will be sent in cleartext\n")
	}

	cfg := &Config{
		PortainerURL: strings.TrimRight(portainerURL, "/"),
		APIKey:       apiKey,
		GitRepo:        os.Getenv("PORTCTL_GIT_REPO"),
		GitBranch:      os.Getenv("PORTCTL_GIT_BRANCH"),
		RepoPath:       os.Getenv("PORTCTL_REPO_PATH"),
		FilesystemPath: os.Getenv("PORTCTL_FILESYSTEM_PATH"),
		EnvFile:        os.Getenv("PORTCTL_ENV_FILE"),
	}

	// A relative env file is relative to the repo, like the stack directories
	if cfg.EnvFile != "" && !filepath.IsAbs(cfg.EnvFile) && cfg.RepoPath != "" {
		cfg.EnvFile = filepath.Join(cfg.RepoPath, cfg.EnvFile)
	}

	if cfg.GitBranch == "" {
		cfg.GitBranch = "main"
	}

	if v := os.Getenv("PORTCTL_ENDPOINT_ID"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("PORTCTL_ENDPOINT_ID must be a numeric ID, got %q", v)
		}
		cfg.EndpointID = id
	}

	if v := os.Getenv("PORTCTL_GIT_CREDENTIAL_ID"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("PORTCTL_GIT_CREDENTIAL_ID must be a numeric ID, got %q", v)
		}
		cfg.GitCredentialID = id
	}

	return cfg, nil
}
