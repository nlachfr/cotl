package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/trace"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

type spanTime struct {
	root *uint64
}

func (v *spanTime) String() string {
	return time.Unix(0, int64(*v.root)).String()
}

func (v *spanTime) Set(s string) error {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*v.root = uint64(t.UnixNano())
	return nil
}

func (v *spanTime) Type() string {
	return "spanTime"
}

type spanAttributes struct {
	attrs *[]*commonv1.KeyValue
}

func (a *spanAttributes) String() string { return fmt.Sprintf("%v", a.attrs) }
func (a *spanAttributes) Set(s string) error {
	for _, part := range strings.Split(s, ",") {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return fmt.Errorf("typo error")
		}
		*a.attrs = append(*a.attrs, &commonv1.KeyValue{
			Key:   kv[0],
			Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: kv[1]}},
		})
	}
	return nil
}
func (a *spanAttributes) Type() string { return "attributes" }

func generateSpanCommandParams() (*tracev1.Span, *pflag.FlagSet) {
	span := &tracev1.Span{}
	opts := pflag.NewFlagSet("span", pflag.ExitOnError)
	opts.BytesHexVar(&span.TraceId, "trace_id", nil, "A unique identifier for a trace")
	opts.BytesHexVar(&span.SpanId, "span_id", nil, "A unique identifier for a span within a trace")
	opts.StringVar(&span.TraceState, "trace_state", "", "")
	opts.BytesHexVar(&span.ParentSpanId, "parent_span_id", nil, "The span_id of this span's parent span")
	opts.StringVar(&span.Name, "name", "", "A description of a span's operation")
	opts.Var(&spanTime{&span.StartTimeUnixNano}, "start_time", "Start time of the span")
	opts.Var(&spanTime{&span.StartTimeUnixNano}, "end_time", "End time of the span")
	opts.Var(&spanAttributes{attrs: &span.Attributes}, "attributes", "Collection of key/value pairs")
	return span, opts
}

func preRunSpanCommand(span *tracev1.Span) error {
	if len(span.TraceId) == 0 {
		id := trace.TraceID{}
		if _, err := rand.Read(id[:]); err != nil {
			return err
		}
		span.TraceId = id[:]
	}
	if len(span.SpanId) == 0 {
		id := trace.SpanID{}
		if _, err := rand.Read(id[:]); err != nil {
			return err
		}
		span.SpanId = id[:]
	}
	if span.StartTimeUnixNano == 0 {
		span.StartTimeUnixNano = uint64(time.Now().UnixNano())
	}
	return nil
}

func runSpanCommand(span *tracev1.Span) error {
	pipedSpan := &tracev1.Span{}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		input, err := io.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			return err
		} else if data, err := base64.RawStdEncoding.DecodeString(string(input)); err != nil {
			return err
		} else if err := proto.Unmarshal(data, pipedSpan); err != nil {
			return err
		}
	}
	proto.Merge(span, pipedSpan)
	if span.Name == "" {
		return fmt.Errorf("A name is required")
	}
	if data, err := proto.Marshal(span); err != nil {
		return err
	} else {
		fmt.Println(base64.RawStdEncoding.EncodeToString(data))
	}
	return nil
}

func newSpanCommand() *cobra.Command {
	cfg, flags := generateSpanCommandParams()
	cmd := &cobra.Command{
		Use:   "span",
		Short: "Create and update your spans on the fly",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}
			return preRunSpanCommand(cfg)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpanCommand(cfg)
		},
	}
	cmd.Flags().AddFlagSet(flags)
	return cmd
}
