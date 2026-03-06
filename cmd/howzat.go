
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var howzatCmd = &cobra.Command{
	Use:   "howzat",
	Short: "Launch the Cricket live tracker",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🏏 Initializing Cricket Tracker...")
	},
}

func init() {
	rootCmd.AddCommand(howzatCmd)
}