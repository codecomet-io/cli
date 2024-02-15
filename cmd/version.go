package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var CurrentVersion = "unknown"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the CodeComet CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(CurrentVersion)
	},
}
