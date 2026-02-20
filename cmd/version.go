package cmd

import (
	"fmt"

	"github.com/longkey1/gotion/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(version.GetFull())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
