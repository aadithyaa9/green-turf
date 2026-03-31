// cmd/f1.go
package cmd

import (
	"fmt"
	"os"

	"github.com/aadithyaa9/green-turf/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var f1Cmd = &cobra.Command{
	Use:   "f1",
	Short: "Launch the Formula 1 live tracker",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewF1Model())
		
		if _, err := p.Run(); err != nil {
			fmt.Printf("Pit lane speed limiter error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(f1Cmd)
}