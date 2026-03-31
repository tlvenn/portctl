package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/tlvenn/portctl/internal/client"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stacks with status, image freshness, and deployed version",
	RunE: func(cmd *cobra.Command, args []string) error {
		stacks, err := api.ListStacks()
		if err != nil {
			return err
		}

		if len(stacks) == 0 {
			fmt.Println("No stacks found.")
			return nil
		}

		// Pre-load images once per endpoint to avoid repeated calls
		imageCache := map[int][]client.Image{}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tIMAGES\tCONTAINERS\tUPDATED\tVERSION")

		for _, s := range stacks {
			containers, err := api.ListContainers(s.EndpointID, s.Name)
			if err != nil {
				fmt.Fprintf(w, "%s\t%s\t-\t(error)\t-\t-\n", s.Name, s.StatusLabel())
				continue
			}

			running, stopped, errored := 0, 0, 0
			for _, ct := range containers {
				switch ct.State {
				case "running":
					running++
				case "exited", "dead":
					stopped++
				default:
					errored++
				}
			}

			// Load images for this endpoint if not cached
			if _, ok := imageCache[s.EndpointID]; !ok {
				imgs, err := api.ListImages(s.EndpointID)
				if err != nil {
					imageCache[s.EndpointID] = nil
				} else {
					imageCache[s.EndpointID] = imgs
				}
			}

			// Check image status for running containers
			images := imageCache[s.EndpointID]
			imageStatus := checkStackImages(s.EndpointID, containers, images)

			// Format update time
			updated := "-"
			if s.UpdateDate > 0 {
				updated = time.Unix(s.UpdateDate, 0).Format("2006-01-02 15:04")
			}

			// Git commit hash (short)
			version := "-"
			if s.GitConfig != nil && s.GitConfig.ConfigHash != "" {
				hash := s.GitConfig.ConfigHash
				if len(hash) > 7 {
					hash = hash[:7]
				}
				version = hash
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d running / %d stopped / %d error\t%s\t%s\n",
				s.Name, s.StatusLabel(), imageStatus, running, stopped, errored, updated, version)
		}

		return w.Flush()
	},
}

func checkStackImages(endpointID int, containers []client.Container, images []client.Image) string {
	if images == nil {
		return "-"
	}

	hasOutdated := false
	allUnknown := true

	for _, ct := range containers {
		if ct.State != "running" {
			continue
		}
		status := api.CheckImageStatus(endpointID, ct, images)
		switch status {
		case client.ImageOutdated:
			hasOutdated = true
			allUnknown = false
		case client.ImageUpToDate:
			allUnknown = false
		}
	}

	if allUnknown {
		return "-"
	}
	if hasOutdated {
		return "outdated"
	}
	return "up to date"
}
