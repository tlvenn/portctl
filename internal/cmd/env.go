package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tlvenn/portctl/internal/client"
)

func init() {
	rootCmd.AddCommand(envCmd)
}

var envCmd = &cobra.Command{
	Use:   "env <stack-name> [set KEY=VALUE... | unset KEY...]",
	Short: "View or modify environment variables for a stack",
	Long: `View, set, or unset environment variables for a Portainer stack.

  portctl env homepage              # List all env vars
  portctl env homepage set K1=V1    # Set one or more vars
  portctl env homepage unset K1     # Remove one or more vars`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]

		stack, err := api.FindStackByName(stackName)
		if err != nil {
			return err
		}
		if stack == nil {
			return fmt.Errorf("stack %q not found", stackName)
		}

		detail, err := api.GetStack(stack.ID)
		if err != nil {
			return err
		}

		// No subcommand — list env vars
		if len(args) == 1 {
			if len(detail.Env) == 0 {
				fmt.Printf("No environment variables set for stack '%s'.\n", stackName)
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tVALUE")
			for _, e := range detail.Env {
				fmt.Fprintf(w, "%s\t%s\n", e.Name, e.Value)
			}
			return w.Flush()
		}

		action := args[1]
		if len(args) < 3 {
			return fmt.Errorf("usage: portctl env %s %s KEY=VALUE...", stackName, action)
		}

		envMap := make(map[string]string)
		for _, e := range detail.Env {
			envMap[e.Name] = e.Value
		}

		switch action {
		case "set":
			for _, kv := range args[2:] {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format %q, expected KEY=VALUE", kv)
				}
				envMap[parts[0]] = parts[1]
			}

		case "unset":
			for _, key := range args[2:] {
				if _, ok := envMap[key]; !ok {
					fmt.Fprintf(os.Stderr, "Warning: %q not found in stack env\n", key)
				}
				delete(envMap, key)
			}

		default:
			return fmt.Errorf("unknown action %q, expected 'set' or 'unset'", action)
		}

		// Build updated env list
		var envList []client.EnvVar
		for k, v := range envMap {
			envList = append(envList, client.EnvVar{Name: k, Value: v})
		}

		payload := client.UpdateStackPayload{
			Env:       envList,
			Prune:     false,
			PullImage: false,
		}

		// Preserve git config for the update
		if detail.GitConfig != nil {
			payload.RepositoryReferenceName = detail.GitConfig.ReferenceName
		}
		if cfg.GitCredentialID != 0 {
			payload.RepositoryAuthentication = true
			payload.RepositoryGitCredentialID = cfg.GitCredentialID
		}

		if err := api.UpdateStackEnv(stack.ID, stack.EndpointID, payload); err != nil {
			return err
		}

		fmt.Printf("Stack '%s' environment updated. Redeploy to apply: portctl redeploy %s\n", stackName, stackName)
		return nil
	},
}
