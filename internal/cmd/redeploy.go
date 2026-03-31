package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tlvenn/portctl/internal/client"
)

func init() {
	rootCmd.AddCommand(redeployCmd)
}

var redeployCmd = &cobra.Command{
	Use:   "redeploy <stack-name>",
	Short: "Trigger a git pull and redeploy for an existing stack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]

		stack, err := api.FindStackByName(stackName)
		if err != nil {
			return err
		}
		if stack == nil {
			return fmt.Errorf("stack %q not found", stackName)
		}

		// Fetch current stack detail to preserve env vars
		detail, err := api.GetStack(stack.ID)
		if err != nil {
			return err
		}

		payload := client.RedeployPayload{
			RepositoryReferenceName: "refs/heads/" + cfg.GitBranch,
			PullImage:               true,
			Env:                     detail.Env,
		}

		// Include git credentials if configured
		if cfg.GitCredentialID != 0 {
			payload.RepositoryAuthentication = true
			payload.RepositoryGitCredentialID = cfg.GitCredentialID
		}

		if err := api.RedeployStack(stack.ID, stack.EndpointID, payload); err != nil {
			return err
		}

		fmt.Printf("Stack '%s' redeployment triggered.\n", stackName)
		return nil
	},
}
