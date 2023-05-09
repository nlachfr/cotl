package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

type TraceParent struct {
	Version    byte
	TraceID    [16]byte
	ParentID   [8]byte
	TraceFlags byte
	valid      bool
}

func (t *TraceParent) String() string {
	return fmt.Sprintf("%s-%s-%s-%s",
		hex.EncodeToString([]byte{t.Version}),
		hex.EncodeToString(t.TraceID[:]),
		hex.EncodeToString(t.ParentID[:]),
		hex.EncodeToString([]byte{t.TraceFlags}),
	)
}

func (t *TraceParent) Set(s string) error {
	parts := strings.Split(s, "-")
	if len(parts) != 4 {
		return fmt.Errorf("invalid traceparent")
	}
	if version, err := hex.DecodeString(parts[0]); err != nil {
		return err
	} else if len(version) != 1 {
		return fmt.Errorf("invalid version: %s", parts[0])
	} else {
		t.Version = version[0]
	}
	if traceId, err := hex.DecodeString(parts[1]); err != nil {
		return err
	} else if len(traceId) != 16 {
		return fmt.Errorf("invalid traceid: %s", parts[1])
	} else {
		t.TraceID = [16]byte(traceId)
	}
	if parentId, err := hex.DecodeString(parts[2]); err != nil {
		return err
	} else if len(parentId) != 8 {
		return fmt.Errorf("invalid parentid: %s", parts[2])
	} else {
		t.ParentID = [8]byte(parentId)
	}
	if flags, err := hex.DecodeString(parts[3]); err != nil {
		return err
	} else if len(flags) != 1 {
		return fmt.Errorf("invalid traceflags: %s", parts[3])
	} else {
		t.TraceFlags = flags[0]
	}
	t.valid = true
	return nil
}

func (v *TraceParent) Type() string {
	return "TraceParent"
}

func (t *TraceParent) IsValid() bool {
	return t.valid
}

type SpanTime struct {
	unixTime uint64
}

func (v *SpanTime) String() string {
	return time.Unix(0, int64(v.unixTime)).String()
}

func (v *SpanTime) Set(s string) error {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	v.unixTime = uint64(t.UnixNano())
	return nil
}

func (v *SpanTime) Type() string {
	return "SpanTime"
}

type SpanAttributes struct {
	attrs []*commonv1.KeyValue
}

func (a *SpanAttributes) String() string { return fmt.Sprintf("%v", a.attrs) }
func (a *SpanAttributes) Set(s string) error {
	for _, part := range strings.Split(s, ",") {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return fmt.Errorf("typo error")
		}
		a.attrs = append(a.attrs, &commonv1.KeyValue{
			Key:   kv[0],
			Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: kv[1]}},
		})
	}
	return nil
}
func (a *SpanAttributes) Type() string { return "SpanAttributes" }

type StatusCode uint64

func (m *StatusCode) String() string { return string(*m) }
func (m *StatusCode) Set(s string) error {
	i, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return err
	}
	switch i {
	case uint64(StatusCodeUnset), uint64(StatusCodeOk), uint64(StatusCodeError):
		*m = StatusCode(i)
	default:
		return fmt.Errorf("invalid status code: %s", s)
	}
	return nil
}
func (m *StatusCode) Type() string { return "statusCode" }

const (
	StatusCodeUnset StatusCode = 0
	StatusCodeOk    StatusCode = 1
	StatusCodeError StatusCode = 2
)

type SpanConfig struct {
	TraceID     []byte
	SpanID      []byte
	TraceState  string
	TraceParent TraceParent
	Name        string
	StartTime   SpanTime
	EndTime     SpanTime
	Attributes  SpanAttributes
	Status      struct {
		Code        StatusCode
		Description string
	}
	BaseSpan *v1.Span
}

func NewSpan(ctx context.Context, cfg *SpanConfig) (*v1.Span, error) {
	if cfg.BaseSpan == nil {
		cfg.BaseSpan = &v1.Span{}
	}
	span := &v1.Span{}
	if len(cfg.TraceID) == 0 {
		if cfg.TraceParent.IsValid() {
			span.TraceId = cfg.TraceParent.TraceID[:]
			span.ParentSpanId = cfg.TraceParent.ParentID[:]
		} else {
			id := trace.TraceID{}
			if _, err := rand.Read(id[:]); err != nil {
				return nil, err
			}
			span.TraceId = id[:]
		}
	}
	if len(cfg.SpanID) == 0 {
		id := trace.SpanID{}
		if _, err := rand.Read(id[:]); err != nil {
			return nil, err
		}
		span.SpanId = id[:]
	}
	if cfg.StartTime.unixTime == 0 {
		span.StartTimeUnixNano = uint64(time.Now().UnixNano())
	}
	code := v1.Status_STATUS_CODE_UNSET
	switch cfg.Status.Code {
	case StatusCodeOk:
		code = v1.Status_STATUS_CODE_OK
	case StatusCodeError:
		code = v1.Status_STATUS_CODE_ERROR
	}
	proto.Merge(span, cfg.BaseSpan)
	proto.Merge(span, &v1.Span{
		TraceState:        cfg.TraceState,
		Name:              cfg.Name,
		StartTimeUnixNano: cfg.StartTime.unixTime,
		EndTimeUnixNano:   cfg.EndTime.unixTime,
		Attributes:        cfg.Attributes.attrs,
		Status: &v1.Status{
			Message: cfg.Status.Description,
			Code:    code,
		},
	})
	return span, nil
}
