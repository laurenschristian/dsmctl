package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
		Use:   "login [otp]",
		Short: "Authenticate and cache a session (prompts for a 2-factor code if needed)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if otp == "" && len(args) == 1 {
				otp = args[0] // allow `dsmctl login 123456`
			}
			otp = strings.TrimSpace(otp)

			sess, err := client.Login(cmd.Context(), account, password, otp)
			// On a 2-factor prompt (or a wrong code), ask for the code and retry.
			for attempts := 0; err != nil && needsOTP(err) && attempts < 3; attempts++ {
				code := promptLine("2-factor code: ")
				if code == "" {
					break
				}
				sess, err = client.Login(cmd.Context(), account, password, code)
			}
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
	cmd.Flags().StringVar(&otp, "otp", "", "2-factor authentication code (also accepted as a positional arg)")
	return cmd
}

// needsOTP reports whether a login error is the 2-factor challenge (403) or a
// failed/expired code (404), i.e. a prompt-and-retry is worth it.
func needsOTP(err error) bool {
	var e *dsm.APIError
	if !errorsAs(err, &e) || e.API != "SYNO.API.Auth" {
		return false
	}
	return e.Code == 403 || e.Code == 404 || e.Code == 406
}

func promptLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return ""
	}
	return strings.TrimSpace(sc.Text())
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
	var watch int
	cmd := &cobra.Command{
		Use:   "health",
		Short: "CPU and memory load, plus volume and disk SMART summary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if watch <= 0 || flagJSON {
				return renderHealth(cmd)
			}
			for {
				fmt.Print("\033[H\033[2J") // clear screen, cursor home
				fmt.Printf("%s  (every %ds, Ctrl-C to stop)\n\n", time.Now().Format("15:04:05"), watch)
				if err := renderHealth(cmd); err != nil {
					return err
				}
				select {
				case <-cmd.Context().Done():
					return nil
				case <-time.After(time.Duration(watch) * time.Second):
				}
			}
		},
	}
	cmd.Flags().IntVar(&watch, "watch", 0, "refresh every N seconds until Ctrl-C")
	return cmd
}

func renderHealth(cmd *cobra.Command) error {
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
	fmt.Printf("mem usage  %s%% of %s\n", u.Memory.RealUsage, humanBytesKB(u.Memory.TotalReal))
	for _, v := range st.Volumes {
		used, total := numInt(v.Size.Used), numInt(v.Size.Total)
		fmt.Printf("volume %-9s %-7s %s / %s (%s)\n", v.ID, v.Status, humanI(used), humanI(total), pct(used, total))
	}
	for _, d := range st.Disks {
		fmt.Printf("disk %-9s %-22s %-7s SMART %-8s %d C\n", d.ID, d.Model, d.Status, d.SMARTStatus, d.Temp)
	}
	return nil
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
	c := rawListCmd("packages", "Installed packages (subcommands: start|stop <id>)", func(cmd *cobra.Command) (any, error) {
		b, err := client.Packages(cmd.Context())
		return rawResult{b}, err
	})
	for _, action := range []string{"start", "stop"} {
		c.AddCommand(packageActionCmd(action))
	}
	return c
}

func packageActionCmd(action string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   action + " <id>",
		Short: action + " an installed package by id (id from `dsmctl packages`)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !confirm(yes, fmt.Sprintf("%s package %q?", action, id)) {
				return fmt.Errorf("aborted (pass --yes to confirm)")
			}
			if err := authed(cmd.Context(), func() error {
				return client.PackageAction(cmd.Context(), action, id)
			}); err != nil {
				return err
			}
			fmt.Printf("%s: %s\n", id, action)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

func dockerCmd() *cobra.Command {
	c := rawListCmd("docker", "Docker containers (subcommands: start|stop|restart)", func(cmd *cobra.Command) (any, error) {
		b, err := client.DockerContainers(cmd.Context())
		return rawResult{b}, err
	})
	c.Aliases = []string{"containers", "ps"}
	for _, action := range []string{"start", "stop", "restart"} {
		c.AddCommand(dockerActionCmd(action))
	}
	return c
}

func dockerActionCmd(action string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   action + " <container>",
		Short: action + " a Docker container by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !confirm(yes, fmt.Sprintf("%s container %q?", action, name)) {
				return fmt.Errorf("aborted (pass --yes to confirm)")
			}
			if err := authed(cmd.Context(), func() error {
				return client.DockerAction(cmd.Context(), action, name)
			}); err != nil {
				return err
			}
			fmt.Printf("%s: %s\n", name, action)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

func tasksCmd() *cobra.Command {
	c := rawListCmd("tasks", "Scheduled tasks (subcommand: run <id>)", func(cmd *cobra.Command) (any, error) {
		b, err := client.Tasks(cmd.Context())
		return rawResult{b}, err
	})
	c.AddCommand(taskRunCmd())
	return c
}

func taskRunCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "run <id>",
		Short: "Run a scheduled task now (id from `dsmctl tasks`)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !confirm(yes, fmt.Sprintf("run task %s now?", id)) {
				return fmt.Errorf("aborted (pass --yes to confirm)")
			}
			if err := authed(cmd.Context(), func() error {
				return client.RunTask(cmd.Context(), id)
			}); err != nil {
				return err
			}
			fmt.Printf("task %s: started\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
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

func errorsAs(err error, target any) bool { return errors.As(err, target) }

// bytes reads a json.Number that may be an int or float string.
func numInt(n json.Number) int64 {
	if v, err := n.Int64(); err == nil {
		return v
	}
	f, _ := n.Float64()
	return int64(f)
}

// humanI formats a byte count with binary units (KiB..PiB).
func humanI(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTP"[exp])
}

// humanBytesKB formats a value already expressed in kibibytes (DSM memory fields).
func humanBytesKB(n json.Number) string { return humanI(numInt(n) * 1024) }

func pct(used, total int64) string {
	if total <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", float64(used)/float64(total)*100)
}
