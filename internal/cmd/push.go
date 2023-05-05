package cmd

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

type spanIDGenerator struct {
	*tracev1.Span
}

func (g *spanIDGenerator) NewIDs(context.Context) (trace.TraceID, trace.SpanID) {
	return trace.TraceID(g.TraceId), trace.SpanID(g.SpanId)
}
func (g *spanIDGenerator) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	return trace.SpanID(g.SpanId)
}

func generatePushCommandParams() *pflag.FlagSet {
	opts := pflag.NewFlagSet("push", pflag.ExitOnError)
	return opts
}

func runPushCommand(ctx context.Context) error {
	rawSpan := &tracev1.Span{}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		input, err := io.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			return err
		} else if data, err := base64.RawStdEncoding.DecodeString(string(input)); err != nil {
			return err
		} else if err := proto.Unmarshal(data, rawSpan); err != nil {
			return err
		}
	}
	var err error
	var exporter sdktrace.SpanExporter
	if exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint()); err != nil {
		return err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithIDGenerator(&spanIDGenerator{rawSpan}),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	if len(rawSpan.ParentSpanId) > 0 {
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID(rawSpan.TraceId),
			SpanID:     trace.SpanID(rawSpan.ParentSpanId),
			TraceFlags: 01,
		})
		ctx = trace.ContextWithRemoteSpanContext(ctx, spanContext)
	}
	_, span := tp.Tracer("").Start(ctx,
		rawSpan.Name,
		trace.WithSpanKind(trace.SpanKind(rawSpan.Kind)),
		trace.WithTimestamp(time.Unix(0, int64(rawSpan.StartTimeUnixNano))),
	)
	if rawSpan.EndTimeUnixNano != 0 {
		span.End(trace.WithTimestamp(time.Unix(0, int64(rawSpan.EndTimeUnixNano))))
	} else {
		span.End()
	}
	return tp.ForceFlush(ctx)
}

func newPushCommand() *cobra.Command {
	flags := generatePushCommandParams()
	cmd := &cobra.Command{
		Use:   "push",
		Short: "End and push your spans directly to the provided backend",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPushCommand(cmd.Context())
		},
	}
	cmd.Flags().AddFlagSet(flags)
	return cmd
}
