// cmd/golazo.go
package cmd

import (
	"fmt"
	"time"

	"github.com/aadithyaa9/green-turf/internal/api"
	"github.com/spf13/cobra"
)

var golazoCmd = &cobra.Command{
	Use:   "golazo",
	Short: "Launch the Football live tracker",
	Run: func(cmd *cobra.Command, args []string) {
		// Get today's date in YYYYMMDD format
		today := time.Now().Format("20060102")
		fmt.Printf("⚽ Fetching live football data for %s...\n", today)

		leagues, err := api.FetchFootballMatches(today)
		if err != nil {
			fmt.Println("❌ Error fetching data:", err)
			return
		}

		fmt.Printf("✅ Successfully fetched %d leagues!\n\n", len(leagues))

		// Let's print the first match of the first league just to prove we have the data
		if len(leagues) > 0 && len(leagues[0].Matches) > 0 {
			firstLeague := leagues[0]
			firstMatch := firstLeague.Matches[0]
			
			fmt.Printf("🏆 %s\n", firstLeague.Name)
			fmt.Printf("🏟️  %s %d - %d %s (Status: %s)\n", 
				firstMatch.Home.Name, 
				firstMatch.Home.Score, 
				firstMatch.Away.Score, 
				firstMatch.Away.Name,
				firstMatch.Status.Reason.Short,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(golazoCmd)
}