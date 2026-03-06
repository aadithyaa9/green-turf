package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "green-turf",
	Short: "A live TUI for football and cricket",
	Long:  `Green Turf is a concurrent terminal dashboard for tracking live football and cricket scores.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Green Turf! Use subcommands 'golazo' or 'howzat'.")
	},
}
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}