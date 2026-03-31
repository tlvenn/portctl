package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tlvenn/portctl/internal/client"
	"github.com/tlvenn/portctl/internal/config"
)

var (
	cfg    *config.Config
	api    *client.Client
)

var rootCmd = &cobra.Command{
	Use:   "portctl",
	Short: "Manage Portainer stacks",
	Long:  "A CLI tool for deploying and managing Docker stacks via the Portainer EE API.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for help and version commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}

		api = client.New(cfg.PortainerURL, cfg.APIKey)
		return nil
	},
}

func Execute() {
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
