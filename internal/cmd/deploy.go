package cmd

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tlvenn/portctl/internal/client"
	"github.com/tlvenn/portctl/internal/webhooks"
)

func init() {
	rootCmd.AddCommand(deployCmd)
}

var validStackName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

var deployCmd = &cobra.Command{
	Use:   "deploy <stack-name>",
	Short: "Deploy a new git-backed stack to Portainer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]

		// Validate stack name
		if !validStackName.MatchString(stackName) {
			return fmt.Errorf("invalid stack name %q: must be alphanumeric with hyphens and underscores only", stackName)
		}

		// Validate git repo is configured
		if cfg.GitRepo == "" {
			return fmt.Errorf("PORTCTL_GIT_REPO is required for deploy")
		}

		// Warn if SSH repo without credential
		if cfg.GitCredentialID == 0 && (strings.HasPrefix(cfg.GitRepo, "ssh://") || strings.HasPrefix(cfg.GitRepo, "git@")) {
			return fmt.Errorf("PORTCTL_GIT_CREDENTIAL_ID is required: %s uses SSH which requires a git credential", cfg.GitRepo)
		}

		// Validate local compose file exists
		if cfg.RepoPath != "" {
			composePath := filepath.Join(cfg.RepoPath, stackName, "docker-compose.yml")
			if _, err := os.Stat(composePath); os.IsNotExist(err) {
				return fmt.Errorf("compose file not found: %s\nMake sure the stack directory exists in the repo", composePath)
			}
		}

		// Validate repo path is writable for webhook registration
		if cfg.RepoPath != "" {
			if info, err := os.Stat(cfg.RepoPath); err != nil || !info.IsDir() {
				return fmt.Errorf("PORTCTL_REPO_PATH %q is not a valid directory", cfg.RepoPath)
			}
		}

		// Check if stack already exists
		existing, err := api.FindStackByName(stackName)
		if err != nil {
			return err
		}
		if existing != nil {
			return fmt.Errorf("stack %q already exists in Portainer (ID: %d)", stackName, existing.ID)
		}

		// Resolve endpoint
		endpointID, err := api.ResolveEndpointID(cfg.EndpointID)
		if err != nil {
			return err
		}

		// Generate webhook UUID
		webhookID, err := generateUUID()
		if err != nil {
			return fmt.Errorf("failed to generate webhook UUID: %w", err)
		}

		// Build payload
		payload := client.CreateStackPayload{
			Name:                    stackName,
			RepositoryURL:          cfg.GitRepo,
			RepositoryReferenceName: "refs/heads/" + cfg.GitBranch,
			ComposeFile:            stackName + "/docker-compose.yml",
			AutoUpdate:             &client.AutoUpdate{Webhook: webhookID},
			SupportRelativePath:    true,
		}

		if cfg.GitCredentialID != 0 {
			payload.RepositoryAuthentication = true
			payload.RepositoryGitCredentialID = cfg.GitCredentialID
		}

		// Create the stack
		stack, err := api.CreateStack(endpointID, payload)
		if err != nil {
			return err
		}

		fmt.Printf("Stack '%s' deployed successfully (ID: %d).\n", stack.Name, stack.ID)
		fmt.Printf("Webhook ID: %s\n", webhookID)

		// Register webhook in portainer-webhooks.json
		if cfg.RepoPath != "" {
			if err := webhooks.Register(cfg.RepoPath, stackName, webhookID); err != nil {
				// Stack was created but webhook registration failed — print recovery info
				fmt.Fprintf(os.Stderr, "\nWarning: failed to update %s: %v\n", webhooks.FileName, err)
				fmt.Fprintf(os.Stderr, "Recovery: manually add %q: [\"%s\"] to %s\n",
					stackName, webhookID, webhooks.FileName)
				return nil
			}
			fmt.Printf("Webhook registered in %s\n", webhooks.FileName)
		} else {
			fmt.Println("Tip: set PORTCTL_REPO_PATH to auto-register webhooks in portainer-webhooks.json")
		}

		return nil
	},
}

func generateUUID() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}
	// Set version 4 and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}
