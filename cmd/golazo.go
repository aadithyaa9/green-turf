// cmd/golazo.go
package cmd

import (
	"fmt"
	"os"

	"github.com/aadithyaa9/green-turf/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var golazoCmd = &cobra.Command{
	Use:   "golazo",
	Short: "Launch the Football live tracker",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewFootballModel())
		
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error starting the UI: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(golazoCmd)
}