package cmd

import "github.com/spf13/cobra"

func BuildCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "cotl",
	}
	root.AddCommand(
		newExtractCommand(),
		newPushCommand(),
		newSpanCommand(),
	)
	return root
}
