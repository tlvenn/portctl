package cmd

import (
	"fmt"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/tlvenn/portctl/internal/client"
)

var noImages bool

func init() {
	listCmd.Flags().BoolVar(&noImages, "no-images", false, "Skip image freshness check (faster)")
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

		// Collect stack info in parallel
		type stackInfo struct {
			stack       client.Stack
			containers  []client.Container
			imageStatus string
			err         error
		}

		infos := make([]stackInfo, len(stacks))
		imageCache := sync.Map{}

		// Phase 1: Fetch containers for all stacks in parallel
		var wg sync.WaitGroup
		for i, s := range stacks {
			infos[i].stack = s
			wg.Add(1)
			go func(idx int, stack client.Stack) {
				defer wg.Done()
				containers, err := api.ListContainers(stack.EndpointID, stack.Name)
				infos[idx].containers = containers
				infos[idx].err = err

				// Pre-load images per endpoint (once)
				if !noImages && err == nil {
					if _, loaded := imageCache.LoadOrStore(stack.EndpointID, nil); !loaded {
						imgs, _ := api.ListImages(stack.EndpointID)
						imageCache.Store(stack.EndpointID, imgs)
					}
				}
			}(i, s)
		}
		wg.Wait()

		// Phase 2: Check image freshness in parallel (if enabled)
		if !noImages {
			// Deduplicate image refs across all containers to avoid checking the same image twice
			type digestResult struct {
				digest string
				err    error
			}
			digestCache := sync.Map{}

			var imgWg sync.WaitGroup
			for i := range infos {
				if infos[i].err != nil {
					continue
				}
				for _, ct := range infos[i].containers {
					if ct.State != "running" || ct.Image == "" {
						continue
					}
					key := fmt.Sprintf("%d:%s", infos[i].stack.EndpointID, ct.Image)
					if _, loaded := digestCache.LoadOrStore(key, (*digestResult)(nil)); !loaded {
						imgWg.Add(1)
						go func(endpointID int, imageRef, cacheKey string) {
							defer imgWg.Done()
							digest, err := api.GetRemoteDigest(endpointID, imageRef)
							digestCache.Store(cacheKey, &digestResult{digest, err})
						}(infos[i].stack.EndpointID, ct.Image, key)
					}
				}
			}
			imgWg.Wait()

			// Phase 3: Compute image status using cached digests
			for i := range infos {
				if infos[i].err != nil {
					infos[i].imageStatus = "-"
					continue
				}

				imgsVal, _ := imageCache.Load(infos[i].stack.EndpointID)
				imgs, _ := imgsVal.([]client.Image)
				if imgs == nil {
					infos[i].imageStatus = "-"
					continue
				}

				hasOutdated := false
				allUnknown := true

				for _, ct := range infos[i].containers {
					if ct.State != "running" || ct.Image == "" {
						continue
					}

					// Find local image
					var localImage *client.Image
					for j := range imgs {
						if imgs[j].ID == ct.ImageID {
							localImage = &imgs[j]
							break
						}
					}
					if localImage == nil {
						continue
					}

					key := fmt.Sprintf("%d:%s", infos[i].stack.EndpointID, ct.Image)
					val, _ := digestCache.Load(key)
					result, _ := val.(*digestResult)
					if result == nil || result.err != nil {
						continue
					}

					allUnknown = false
					matched := false
					for _, d := range localImage.RepoDigests {
						if len(result.digest) > 0 && contains(d, result.digest) {
							matched = true
							break
						}
					}
					if !matched {
						hasOutdated = true
					}
				}

				if allUnknown {
					infos[i].imageStatus = "-"
				} else if hasOutdated {
					infos[i].imageStatus = "outdated"
				} else {
					infos[i].imageStatus = "up to date"
				}
			}
		}

		// Print results
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if noImages {
			fmt.Fprintln(w, "NAME\tSTATUS\tCONTAINERS\tUPDATED\tVERSION")
		} else {
			fmt.Fprintln(w, "NAME\tSTATUS\tIMAGES\tCONTAINERS\tUPDATED\tVERSION")
		}

		for _, info := range infos {
			s := info.stack

			var containerStr string
			if info.err != nil {
				containerStr = "(error)"
			} else {
				running, stopped, errored := 0, 0, 0
				for _, ct := range info.containers {
					switch ct.State {
					case "running":
						running++
					case "exited", "dead":
						stopped++
					default:
						errored++
					}
				}
				containerStr = fmt.Sprintf("%d running / %d stopped / %d error", running, stopped, errored)
			}

			updated := "-"
			if s.UpdateDate > 0 {
				updated = time.Unix(s.UpdateDate, 0).Format("2006-01-02 15:04")
			}

			version := "-"
			if s.GitConfig != nil && s.GitConfig.ConfigHash != "" {
				hash := s.GitConfig.ConfigHash
				if len(hash) > 7 {
					hash = hash[:7]
				}
				version = hash
			}

			if noImages {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					s.Name, s.StatusLabel(), containerStr, updated, version)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					s.Name, s.StatusLabel(), info.imageStatus, containerStr, updated, version)
			}
		}

		return w.Flush()
	},
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && indexString(s, substr) >= 0
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
