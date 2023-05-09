package trace

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

type PushMode string

func (m *PushMode) String() string { return string(*m) }
func (m *PushMode) Set(s string) error {
	switch s {
	case string(PushModeStdout), string(PushModeOtlp), string(PushModeOtlpHttp):
		*m = PushMode(s)
	default:
		return fmt.Errorf("invalid push mode: %s", s)
	}
	return nil
}
func (m *PushMode) Type() string { return "pushMode" }

const (
	PushModeStdout   PushMode = "stdout"
	PushModeOtlp     PushMode = "otlp"
	PushModeOtlpHttp PushMode = "otlphttp"
)

type spanIDGenerator struct {
	*v1.Span
}

func (g *spanIDGenerator) NewIDs(context.Context) (trace.TraceID, trace.SpanID) {
	return trace.TraceID(g.TraceId), trace.SpanID(g.SpanId)
}
func (g *spanIDGenerator) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	return trace.SpanID(g.SpanId)
}

type PushConfig struct {
	Mode PushMode
	Span *v1.Span
}

func Push(ctx context.Context, cfg *PushConfig) error {
	var (
		err      error
		exporter sdktrace.SpanExporter
	)
	switch cfg.Mode {
	case PushModeStdout:
		if exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint()); err != nil {
			return err
		}
	case PushModeOtlp:
		if exporter, err = otlptracegrpc.New(ctx); err != nil {
			return err
		}
	case PushModeOtlpHttp:
		if exporter, err = otlptracehttp.New(ctx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid push mode: %s", cfg.Mode)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithIDGenerator(&spanIDGenerator{cfg.Span}),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	if len(cfg.Span.ParentSpanId) > 0 {
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID(cfg.Span.TraceId),
			SpanID:     trace.SpanID(cfg.Span.ParentSpanId),
			TraceFlags: 01,
		})
		ctx = trace.ContextWithRemoteSpanContext(ctx, spanContext)
	}
	_, span := tp.Tracer("").Start(ctx,
		cfg.Span.Name,
		trace.WithSpanKind(trace.SpanKind(cfg.Span.Kind)),
		trace.WithTimestamp(time.Unix(0, int64(cfg.Span.StartTimeUnixNano))),
	)
	span.SetStatus(codes.Code(cfg.Span.Status.Code), cfg.Span.Status.Message)
	if cfg.Span.EndTimeUnixNano != 0 {
		span.End(trace.WithTimestamp(time.Unix(0, int64(cfg.Span.EndTimeUnixNano))))
	} else {
		span.End()
	}
	return tp.ForceFlush(ctx)
}
