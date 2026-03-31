package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(credentialsCmd)
}

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "List git credentials stored in Portainer",
	RunE: func(cmd *cobra.Command, args []string) error {
		user, err := api.GetCurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current user (this feature requires Portainer EE): %w", err)
		}

		creds, err := api.ListGitCredentials(user.ID)
		if err != nil {
			return fmt.Errorf("failed to list git credentials (this feature requires Portainer EE): %w", err)
		}

		if len(creds) == 0 {
			fmt.Println("No git credentials found.")
			fmt.Println("Add credentials in Portainer: Settings > Git credentials")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME")
		for _, c := range creds {
			fmt.Fprintf(w, "%d\t%s\n", c.ID, c.Name)
		}
		return w.Flush()
	},
}
