package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/version"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the ali version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ali %s\n", version.Version)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", version.Commit)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "built:  %s\n", version.BuildDate)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "author: %s\n", version.Author)
		},
	}

	return cmd
}
