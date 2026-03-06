package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var golazoCmd = &cobra.Command{
	Use:   "golazo",
	Short: "Launch the Football live tracker",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("⚽ Initializing Football Tracker...")
	},
}

func init() {
	rootCmd.AddCommand(golazoCmd)
}