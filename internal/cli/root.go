// Package cli wires the cobra commands.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/dsmctl/internal/config"
	"github.com/laurenschristian/dsmctl/internal/dsm"
)

var (
	Version = "dev"

	flagJSON bool
	cfg      *config.Config
	client   *dsm.Client
)

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "dsmctl",
		Short:         "CLI and MCP server for Synology DSM",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			if cfg, err = config.Load(); err != nil {
				return err
			}
			// init writes the config and needs no URL or client yet.
			if cmd.Name() == "init" {
				return nil
			}
			if cfg.URL == "" {
				return errors.New("no DSM url set: run `dsmctl init` or set DSM_URL")
			}
			client = dsm.New(cfg.URL)
			sess := config.LoadSession()
			client.SID = sess.SID
			client.SynoToken = sess.SynoToken
			client.OnSession = func(sid, synoToken string) error {
				return config.SaveSession(config.Session{SID: sid, SynoToken: synoToken})
			}
			return nil
		},
	}
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "print raw JSON")
	root.AddCommand(
		initCmd(), loginCmd(), logoutCmd(), doctorCmd(),
		statusCmd(), healthCmd(), storageCmd(),
		packagesCmd(), dockerCmd(), tasksCmd(), sharesCmd(), usersCmd(), connectionsCmd(),
		rebootCmd(), shutdownCmd(), rawCmd(), mcpCmd(),
	)
	return root
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// emitRaw pretty-prints a raw JSON payload (or emits it as-is with --json).
func emitRaw(raw json.RawMessage) error {
	if flagJSON {
		_, err := fmt.Fprintln(os.Stdout, string(raw))
		return err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		_, err = fmt.Fprintln(os.Stdout, string(raw))
		return err
	}
	return emit(v)
}

// ensureSession auto-logs-in when there is no session but a password is
// available and 2FA is off. Otherwise it surfaces a clear "run login" error.
func ensureSession(ctx context.Context) error {
	if client.SID != "" {
		return nil
	}
	if cfg.Username == "" {
		return dsm.ErrNotLoggedIn
	}
	if err := cfg.Resolve(); err != nil {
		return err
	}
	if cfg.Password == "" {
		return dsm.ErrNotLoggedIn
	}
	_, err := client.Login(ctx, cfg.Username, cfg.Password, "")
	return err
}

// authed runs fn after ensuring a session, retrying once if the session expired
// mid-call and we can re-login.
func authed(ctx context.Context, fn func() error) error {
	if err := ensureSession(ctx); err != nil {
		return err
	}
	err := fn()
	var apiErr *dsm.APIError
	if errors.As(err, &apiErr) && sessionExpired(apiErr) && cfg.Username != "" && cfg.Password != "" {
		client.SID, client.SynoToken = "", ""
		if _, lerr := client.Login(ctx, cfg.Username, cfg.Password, ""); lerr == nil {
			return fn()
		}
	}
	return err
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config, reachability, and session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			reachable := client.Ping(cmd.Context()) == nil
			sess := config.LoadSession()
			state := map[string]any{
				"config":       config.Path(),
				"url":          client.BaseURL,
				"reachable":    reachable,
				"has_session":  sess.SID != "",
				"username_set": cfg.Username != "",
			}
			if flagJSON {
				return emit(state)
			}
			fmt.Printf("config     %s\n", config.Path())
			fmt.Printf("url        %s\n", client.BaseURL)
			fmt.Printf("reachable  %v\n", reachable)
			fmt.Printf("session    %v\n", sess.SID != "")
			fmt.Printf("username   %v\n", cfg.Username != "")
			return nil
		},
	}
}
