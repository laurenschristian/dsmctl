package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/dsmctl/internal/dsm"
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Check for an available DSM update",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var u *dsm.UpdateInfo
			if err := authed(cmd.Context(), func() (err error) {
				u, err = client.UpdateCheck(cmd.Context())
				return
			}); err != nil {
				return err
			}
			if flagJSON {
				return emit(u)
			}
			if !u.Available {
				fmt.Println("up to date")
				return nil
			}
			fmt.Printf("update available: %s\n", u.Version)
			if u.Reboot != "" && u.Reboot != "none" {
				fmt.Printf("reboot: %s\n", u.Reboot)
			}
			return nil
		},
	}
}

func securityCmd() *cobra.Command {
	return rawListCmd("security", "Security Advisor scan summary", func(cmd *cobra.Command) (any, error) {
		b, err := client.Security(cmd.Context())
		return rawResult{b}, err
	})
}

func networkCmd() *cobra.Command {
	return rawListCmd("network", "Network interfaces, DNS, and gateway", func(cmd *cobra.Command) (any, error) {
		b, err := client.Network(cmd.Context())
		return rawResult{b}, err
	})
}

func upsCmd() *cobra.Command {
	return rawListCmd("ups", "UPS status (model, charge, load, runtime)", func(cmd *cobra.Command) (any, error) {
		b, err := client.UPS(cmd.Context())
		return rawResult{b}, err
	})
}

func processesCmd() *cobra.Command {
	c := rawListCmd("processes", "Running processes with CPU and memory", func(cmd *cobra.Command) (any, error) {
		b, err := client.Processes(cmd.Context())
		return rawResult{b}, err
	})
	c.Aliases = []string{"top"}
	return c
}

func groupsCmd() *cobra.Command {
	return rawListCmd("groups", "Local groups", func(cmd *cobra.Command) (any, error) {
		b, err := client.Groups(cmd.Context())
		return rawResult{b}, err
	})
}

func backupCmd() *cobra.Command {
	return rawListCmd("backup", "Hyper Backup tasks and their state", func(cmd *cobra.Command) (any, error) {
		b, err := client.BackupTasks(cmd.Context())
		return rawResult{b}, err
	})
}

// apisCmd lists every WebAPI the host exposes, optionally filtered by substring,
// so `dsmctl raw` can reach anything not wrapped by a first-class command.
func apisCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apis [filter]",
		Short: "List every DSM WebAPI (optionally filter by substring), for use with raw",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var raw json.RawMessage
			if err := authed(cmd.Context(), func() (err error) {
				raw, err = client.APIs(cmd.Context())
				return
			}); err != nil {
				return err
			}
			var m map[string]struct {
				MinVersion int    `json:"minVersion"`
				MaxVersion int    `json:"maxVersion"`
				Path       string `json:"path"`
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			filter := ""
			if len(args) == 1 {
				filter = strings.ToLower(args[0])
			}
			names := make([]string, 0, len(m))
			for k := range m {
				if filter == "" || strings.Contains(strings.ToLower(k), filter) {
					names = append(names, k)
				}
			}
			sort.Strings(names)
			if flagJSON {
				out := make(map[string]any, len(names))
				for _, n := range names {
					out[n] = m[n]
				}
				return emit(out)
			}
			for _, n := range names {
				fmt.Printf("%-52s v%d-%d\n", n, m[n].MinVersion, m[n].MaxVersion)
			}
			fmt.Printf("\n%d APIs\n", len(names))
			return nil
		},
	}
}
