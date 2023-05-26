package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version string = "0.0.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of zombie-detector",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}
