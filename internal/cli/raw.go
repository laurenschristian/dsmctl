package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

// rawCmd calls any DSM API: dsmctl raw SYNO.Core.System info --version 3 -p key=val
func rawCmd() *cobra.Command {
	var version int
	var params []string
	cmd := &cobra.Command{
		Use:   "raw <api> <method>",
		Short: "Call any DSM API endpoint (SYNO.*), for anything not covered by a command",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := url.Values{}
			for _, kv := range params {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("bad --param %q, want key=value", kv)
				}
				p.Set(k, v)
			}
			var raw []byte
			if err := authed(cmd.Context(), func() error {
				b, err := client.Raw(cmd.Context(), args[0], args[1], version, p)
				raw = b
				return err
			}); err != nil {
				return err
			}
			return emitRaw(raw)
		},
	}
	cmd.Flags().IntVar(&version, "version", 1, "API version")
	cmd.Flags().StringArrayVarP(&params, "param", "p", nil, "extra key=value param (repeatable)")
	return cmd
}
