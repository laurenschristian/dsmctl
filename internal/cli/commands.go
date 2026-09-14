package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/dsmctl/internal/config"
	"github.com/laurenschristian/dsmctl/internal/dsm"
)

func initCmd() *cobra.Command {
	var url, user, pass, passCmd string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a config file (url, username, password source)",
		RunE: func(_ *cobra.Command, _ []string) error {
			c, _ := config.Load()
			if url != "" {
				c.URL = url
			}
			if user != "" {
				c.Username = user
			}
			if pass != "" {
				c.Password = pass
			}
			if passCmd != "" {
				c.PasswordCmd = passCmd
			}
			if err := config.Save(c); err != nil {
				return err
			}
			fmt.Println("wrote", config.Path())
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "DSM base url, e.g. http://10.0.0.2:5000")
	cmd.Flags().StringVar(&user, "user", "", "DSM account")
	cmd.Flags().StringVar(&pass, "password", "", "DSM password (stored plaintext, mode 0600)")
	cmd.Flags().StringVar(&passCmd, "password-cmd", "", "shell command that prints the password")
	return cmd
}

func loginCmd() *cobra.Command {
	var account, password, otp string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate and cache a session (use --otp for 2-factor accounts)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if account == "" {
				account = cfg.Username
			}
			if account == "" {
				return fmt.Errorf("no account: pass --account or set username in config")
			}
			if password == "" {
				if err := cfg.Resolve(); err == nil {
					password = cfg.Password
				}
			}
			if password == "" {
				return fmt.Errorf("no password: pass --password or set password_cmd in config")
			}
			sess, err := client.Login(cmd.Context(), account, password, otp)
			if err != nil {
				return err
			}
			if flagJSON {
				return emit(map[string]any{"logged_in": true, "has_synotoken": sess.SynoToken != ""})
			}
			fmt.Println("logged in, session cached at", config.SessionPath())
			return nil
		},
	}
	cmd.Flags().StringVar(&account, "account", "", "DSM account (defaults to config username)")
	cmd.Flags().StringVar(&password, "password", "", "DSM password (defaults to password_cmd)")
	cmd.Flags().StringVar(&otp, "otp", "", "2-factor authentication code")
	return cmd
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Invalidate and clear the cached session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = client.Logout(cmd.Context())
			if err := config.ClearSession(); err != nil {
				return err
			}
			fmt.Println("logged out")
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Model, DSM version, uptime, temperature",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var s *dsm.SystemInfo
			if err := authed(cmd.Context(), func() (err error) {
				s, err = client.System(cmd.Context())
				return
			}); err != nil {
				return err
			}
			if flagJSON {
				return emit(s)
			}
			fmt.Printf("model    %s\n", s.Model)
			fmt.Printf("version  %s\n", firstNonEmpty(s.Version, s.FirmwareVer))
			fmt.Printf("serial   %s\n", s.Serial)
			fmt.Printf("cpu      %s (%s cores)\n", s.CPUSeries, s.CPUCores)
			fmt.Printf("ram      %d MB\n", s.RAMSize)
			fmt.Printf("uptime   %s\n", uptime(s))
			if s.Temperature > 0 {
				warn := ""
				if s.TempWarn {
					warn = " (WARNING)"
				}
				fmt.Printf("temp     %d C%s\n", s.Temperature, warn)
			}
			return nil
		},
	}
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "CPU and memory load, plus volume and disk SMART summary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var u *dsm.Utilization
			var st *dsm.StorageInfo
			if err := authed(cmd.Context(), func() error {
				var err error
				if u, err = client.Utilization(cmd.Context()); err != nil {
					return err
				}
				st, err = client.Storage(cmd.Context())
				return err
			}); err != nil {
				return err
			}
			if flagJSON {
				return emit(map[string]any{"utilization": u, "storage": st})
			}
			fmt.Printf("cpu load   user %s%% system %s%%\n", u.CPU.UserLoad, u.CPU.SystemLoad)
			fmt.Printf("mem usage  %s%%\n", u.Memory.RealUsage)
			for _, v := range st.Volumes {
				fmt.Printf("volume %-8s %-8s used %s / %s\n", v.ID, v.Status, v.Size.Used, v.Size.Total)
			}
			for _, d := range st.Disks {
				fmt.Printf("disk %-8s %-20s %-6s SMART %-8s %d C\n", d.ID, d.Model, d.Status, d.SMARTStatus, d.Temp)
			}
			return nil
		},
	}
}

func storageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "storage",
		Short: "Volumes and physical disks with SMART status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var st *dsm.StorageInfo
			if err := authed(cmd.Context(), func() (err error) {
				st, err = client.Storage(cmd.Context())
				return
			}); err != nil {
				return err
			}
			return emit(st)
		},
	}
}

// rawListCmd builds a command that dumps a raw endpoint payload.
func rawListCmd(use, short string, fn func(cmd *cobra.Command) (any, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var out any
			if err := authed(cmd.Context(), func() (err error) {
				out, err = fn(cmd)
				return
			}); err != nil {
				return err
			}
			if raw, ok := out.(rawResult); ok {
				return emitRaw(raw.data)
			}
			return emit(out)
		},
	}
}

type rawResult struct{ data []byte }

func packagesCmd() *cobra.Command {
	return rawListCmd("packages", "Installed packages", func(cmd *cobra.Command) (any, error) {
		b, err := client.Packages(cmd.Context())
		return rawResult{b}, err
	})
}

func dockerCmd() *cobra.Command {
	c := rawListCmd("docker", "Docker containers", func(cmd *cobra.Command) (any, error) {
		b, err := client.DockerContainers(cmd.Context())
		return rawResult{b}, err
	})
	c.Aliases = []string{"containers", "ps"}
	return c
}

func tasksCmd() *cobra.Command {
	return rawListCmd("tasks", "Scheduled tasks", func(cmd *cobra.Command) (any, error) {
		b, err := client.Tasks(cmd.Context())
		return rawResult{b}, err
	})
}

func sharesCmd() *cobra.Command {
	return rawListCmd("shares", "Shared folders", func(cmd *cobra.Command) (any, error) {
		b, err := client.Shares(cmd.Context())
		return rawResult{b}, err
	})
}

func usersCmd() *cobra.Command {
	return rawListCmd("users", "Local users", func(cmd *cobra.Command) (any, error) {
		b, err := client.Users(cmd.Context())
		return rawResult{b}, err
	})
}

func connectionsCmd() *cobra.Command {
	c := rawListCmd("connections", "Active service connections", func(cmd *cobra.Command) (any, error) {
		b, err := client.Connections(cmd.Context())
		return rawResult{b}, err
	})
	c.Aliases = []string{"who"}
	return c
}

func rebootCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "reboot",
		Short: "Restart the DSM host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm(yes, "Reboot the NAS?") {
				return fmt.Errorf("aborted (pass --yes to confirm)")
			}
			return authed(cmd.Context(), func() error { return client.Reboot(cmd.Context()) })
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

func shutdownCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "shutdown",
		Short: "Power off the DSM host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm(yes, "Shut down the NAS?") {
				return fmt.Errorf("aborted (pass --yes to confirm)")
			}
			return authed(cmd.Context(), func() error { return client.Shutdown(cmd.Context()) })
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

func confirm(yes bool, prompt string) bool {
	if yes {
		return true
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	a := strings.ToLower(strings.TrimSpace(sc.Text()))
	return a == "y" || a == "yes"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func uptime(s *dsm.SystemInfo) string {
	if s.Uptime != "" {
		return s.Uptime // DSM returns "H:M:S", e.g. 5253:31:53
	}
	return "unknown"
}

func sessionExpired(e *dsm.APIError) bool { return e.SessionExpired() }
