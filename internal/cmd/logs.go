package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

func init() {
	logsCmd.Flags().IntVar(&tailLines, "tail", 100, "Number of lines to show from the end of logs")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	rootCmd.AddCommand(logsCmd)
}

var (
	tailLines int
	follow    bool
)

type logTarget struct {
	id   string
	name string
}

var logsCmd = &cobra.Command{
	Use:   "logs <stack-name> [service]",
	Short: "View container logs for a stack or specific service",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]
		var serviceName string
		if len(args) > 1 {
			serviceName = args[1]
		}

		// Resolve endpoint
		endpointID, err := api.ResolveEndpointID(cfg.EndpointID)
		if err != nil {
			return err
		}

		// List containers for this stack
		containers, err := api.ListContainers(endpointID, stackName)
		if err != nil {
			return err
		}

		if len(containers) == 0 {
			return fmt.Errorf("no containers found for stack %q", stackName)
		}

		// Build log targets
		var targets []logTarget
		var available []string
		for _, ct := range containers {
			svc := ct.ServiceName()
			available = append(available, svc)
			if serviceName == "" || svc == serviceName {
				targets = append(targets, logTarget{ct.ID, svc})
			}
		}

		if serviceName != "" && len(targets) == 0 {
			return fmt.Errorf("service %q not found in stack %q; available services: %v", serviceName, stackName, available)
		}

		// Single container — no prefix needed
		if len(targets) == 1 {
			return streamLogs(endpointID, targets[0].id, "", tailLines, follow)
		}

		// Multiple containers
		if !follow {
			for _, t := range targets {
				if err := streamLogs(endpointID, t.id, t.name, tailLines, false); err != nil {
					fmt.Fprintf(os.Stderr, "Error getting logs for %s: %v\n", t.name, err)
				}
			}
			return nil
		}

		// Follow mode — stream in parallel
		var wg sync.WaitGroup
		for _, t := range targets {
			wg.Add(1)
			go func(id, name string) {
				defer wg.Done()
				if err := streamLogs(endpointID, id, name, tailLines, true); err != nil {
					fmt.Fprintf(os.Stderr, "Error streaming logs for %s: %v\n", name, err)
				}
			}(t.id, t.name)
		}
		wg.Wait()
		return nil
	},
}

func streamLogs(endpointID int, containerID, prefix string, tail int, follow bool) error {
	reader, err := api.GetContainerLogs(endpointID, containerID, tail, follow)
	if err != nil {
		return err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if prefix != "" {
			fmt.Printf("[%s] %s\n", prefix, scanner.Text())
		} else {
			fmt.Println(scanner.Text())
		}
	}
	return scanner.Err()
}
