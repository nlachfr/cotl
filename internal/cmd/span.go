package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/nlachfr/cotl/internal/trace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newSpanCommand() *cobra.Command {
	cfg := &trace.SpanConfig{}
	cmd := &cobra.Command{
		Use:   "span",
		Short: "Create and update your spans on the fly",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				input, err := io.ReadAll(bufio.NewReader(os.Stdin))
				if err != nil {
					return err
				} else if cfg.BaseSpan, err = UnmarshalSpan(string(input)); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if span, err := trace.NewSpan(cmd.Context(), cfg); err != nil {
				return err
			} else if s, err := MarshalSpan(span); err != nil {
				return err
			} else {
				fmt.Println(s)
			}
			return nil
		},
	}
	cmd.Flags().BytesHexVar(&cfg.TraceID, "trace_id", nil, "A unique identifier for a trace")
	cmd.Flags().BytesHexVar(&cfg.SpanID, "span_id", nil, "A unique identifier for a span within a trace")
	cmd.Flags().StringVar(&cfg.TraceState, "trace_state", "", "Extends trace_parent with vendor-specific data")
	cmd.Flags().Var(&cfg.TraceParent, "trace_parent", "Describes the position of the incoming request in its trace graph")
	cmd.Flags().StringVar(&cfg.Name, "name", "", "A description of a span's operation")
	cmd.Flags().Var(&cfg.StartTime, "start_time", "Start time of the span")
	cmd.Flags().Var(&cfg.EndTime, "end_time", "End time of the span")
	cmd.Flags().Var(&cfg.Attributes, "attributes", "Collection of key/value pairs")
	cmd.Flags().Var(&cfg.Status.Code, "status_code", "An optional final status for this span")
	cmd.Flags().StringVar(&cfg.Status.Description, "status_description", "", "A description for the status of this span")
	return cmd
}
