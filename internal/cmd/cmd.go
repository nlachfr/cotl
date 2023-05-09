package cmd

import (
	"encoding/base64"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

func BuildCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "cotl",
	}
	root.AddCommand(
		newPushCommand(),
		newSpanCommand(),
		newTraceparentCommand(),
	)
	return root
}

func MarshalSpan(s *v1.Span) (string, error) {
	if data, err := proto.Marshal(s); err != nil {
		return "", err
	} else {
		return base64.RawStdEncoding.EncodeToString(data), nil
	}
}

func UnmarshalSpan(s string) (*v1.Span, error) {
	span := &v1.Span{}
	if data, err := base64.RawStdEncoding.DecodeString(s); err != nil {
		return nil, err
	} else if err = proto.Unmarshal(data, span); err != nil {
		return nil, err
	}
	return span, nil
}
