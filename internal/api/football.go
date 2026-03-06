// internal/api/football.go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/aadithyaa9/green-turf/internal/models"
)

type espnResponse struct {
	Events []struct {
		Competitions []struct {
			Competitors []struct {
				HomeAway string `json:"homeAway"`
				Score    string `json:"score"`
				Team     struct {
					Name string `json:"name"`
				} `json:"team"`
			} `json:"competitors"`
		} `json:"competitions"`
		Status struct {
			Type struct {
				Detail string `json:"detail"` // e.g., "Full Time", "Halftime"
			} `json:"type"`
		} `json:"status"`
	} `json:"events"`
}

// FetchFootballMatches gets the live English Premier League scoreboard from ESPN
func FetchFootballMatches(date string) ([]models.League, error) {
	// ESPN's public, unblocked endpoint for the Premier League (eng.1)
	url := "https://site.api.espn.com/apis/site/v2/sports/soccer/eng.1/scoreboard"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var espn espnResponse
	if err := json.NewDecoder(resp.Body).Decode(&espn); err != nil {
		return nil, err
	}

	// Now we map ESPN's messy data into our clean Green Turf models!
	var matches []models.Match
	for _, event := range espn.Events {
		if len(event.Competitions) == 0 || len(event.Competitions[0].Competitors) < 2 {
			continue // Skip if data is malformed
		}

		c1 := event.Competitions[0].Competitors[0]
		c2 := event.Competitions[0].Competitors[1]

		// Figure out who is Home and who is Away
		home, away := c1, c2
		if c2.HomeAway == "home" {
			home, away = c2, c1
		}

		// ESPN returns scores as strings ("2"), so we convert them to integers for our UI
		homeScore, _ := strconv.Atoi(home.Score)
		awayScore, _ := strconv.Atoi(away.Score)

		matches = append(matches, models.Match{
			Home: models.Team{Name: home.Team.Name, Score: homeScore},
			Away: models.Team{Name: away.Team.Name, Score: awayScore},
			Status: models.Status{
				Reason: struct {
					Short string `json:"short"`
				}{Short: event.Status.Type.Detail},
			},
		})
	}

	// Wrap the matches in our League struct
	premierLeague := models.League{
		Name:    "English Premier League (ESPN)",
		Matches: matches,
	}

	// We return it exactly how our Bubble Tea UI expects it!
	return []models.League{premierLeague}, nil
}