package cmd

import "github.com/spf13/cobra"

func newExtractCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract information from your spans for environment propagation",
	}
	return cmd
}
