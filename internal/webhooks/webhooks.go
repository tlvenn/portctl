package webhooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const FileName = "portainer-webhooks.json"

func filePath(repoPath string) string {
	return filepath.Join(repoPath, FileName)
}

func Load(repoPath string) (map[string][]string, error) {
	data, err := os.ReadFile(filePath(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]string), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", FileName, err)
	}

	var webhooks map[string][]string
	if err := json.Unmarshal(data, &webhooks); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", FileName, err)
	}
	return webhooks, nil
}

func Save(repoPath string, webhooks map[string][]string) error {
	data, err := json.MarshalIndent(webhooks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal webhooks: %w", err)
	}

	// Append trailing newline for clean git diffs
	data = append(data, '\n')

	if err := os.WriteFile(filePath(repoPath), data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", FileName, err)
	}
	return nil
}

func Register(repoPath, stackName, webhookID string) error {
	webhooks, err := Load(repoPath)
	if err != nil {
		return err
	}

	webhooks[stackName] = append(webhooks[stackName], webhookID)
	return Save(repoPath, webhooks)
}
