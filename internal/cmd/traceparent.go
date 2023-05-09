package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/nlachfr/cotl/internal/trace"
	"github.com/spf13/cobra"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"golang.org/x/term"
)

func newTraceparentCommand() *cobra.Command {
	var span *v1.Span
	cmd := &cobra.Command{
		Use:   "traceparent",
		Short: "Generate W3C traceparent from a given span",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				input, err := io.ReadAll(bufio.NewReader(os.Stdin))
				if err != nil {
					return err
				} else if span, err = UnmarshalSpan(string(input)); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if span == nil {
				return fmt.Errorf("A span is required")
			}
			fmt.Println((&trace.TraceParent{
				Version:    0x00,
				TraceID:    [16]byte(span.TraceId),
				ParentID:   [8]byte(span.SpanId),
				TraceFlags: 0x01,
			}).String())
			return nil
		},
	}
	return cmd
}
