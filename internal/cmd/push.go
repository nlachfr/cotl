package cmd

import (
	"bufio"
	"io"
	"os"

	"github.com/nlachfr/cotl/internal/trace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newPushCommand() *cobra.Command {
	cfg := &trace.PushConfig{}
	cmd := &cobra.Command{
		Use:   "push",
		Short: "End and push your spans directly to the provided backend",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				input, err := io.ReadAll(bufio.NewReader(os.Stdin))
				if err != nil {
					return err
				} else if cfg.Span, err = UnmarshalSpan(string(input)); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return trace.Push(cmd.Context(), cfg)
		},
	}
	cmd.Flags().Var(&cfg.Mode, "exporter", "Configure the exporter used")
	return cmd
}
