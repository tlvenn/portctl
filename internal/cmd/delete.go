package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete <stack-name>",
	Short: "Delete a stack from Portainer",
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

		if err := api.DeleteStack(stack.ID, stack.EndpointID); err != nil {
			return err
		}

		fmt.Printf("Stack '%s' deleted (ID: %d).\n", stackName, stack.ID)
		return nil
	},
}
