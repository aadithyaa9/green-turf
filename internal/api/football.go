// internal/api/football.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync" // NEW: Required for concurrency
	"time"

	"github.com/aadithyaa9/green-turf/internal/models"
)

type espnResponse struct {
	Events []struct {
		ID           string `json:"id"`
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
				Detail string `json:"detail"`
			} `json:"type"`
		} `json:"status"`
	} `json:"events"`
}

type espnSummary struct {
	KeyEvents []struct {
		Type struct {
			Text string `json:"text"`
		} `json:"type"`
		Clock struct {
			DisplayValue string `json:"displayValue"`
		} `json:"clock"`
		Team struct {
			Name string `json:"name"`
		} `json:"team"`
		Participants []struct {
			Athlete struct {
				DisplayName string `json:"displayName"`
			} `json:"athlete"`
		} `json:"participants"`
	} `json:"keyEvents"`
}
func FetchFootballMatches(date string) ([]models.League, error) {
	now := time.Now()
	
	// THE FIX: Widen the window to cover international breaks and off-weeks
	// Look back 14 days, and look forward 3 days
	startDate := now.AddDate(0, 0, -14).Format("20060102")
	endDate := now.AddDate(0, 0, 3).Format("20060102")

	leaguesToFetch := map[string]string{
		"English Premier League": "eng.1",
		"Spanish LALIGA":         "esp.1",
		"Italian Serie A":        "ita.1",
		"German Bundesliga":      "ger.1",
		"French Ligue 1":         "fra.1",
	}

	var allLeagues []models.League
	var mu sync.Mutex
	var wg sync.WaitGroup

	client := &http.Client{Timeout: 10 * time.Second}

	for name, code := range leaguesToFetch {
		wg.Add(1)

		go func(leagueName, leagueCode string) {
			defer wg.Done()

			// Use our new wide date range
			url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/soccer/%s/scoreboard?dates=%s-%s", 
				leagueCode, startDate, endDate)
			
			resp, err := client.Get(url)
			if err != nil {
				return // If one league fails (network blip), just skip it cleanly
			}
			defer resp.Body.Close()

			var espn espnResponse
			if err := json.NewDecoder(resp.Body).Decode(&espn); err != nil {
				return
			}

			var matches []models.Match
			for _, event := range espn.Events {
				if len(event.Competitions) == 0 || len(event.Competitions[0].Competitors) < 2 {
					continue
				}

				c1 := event.Competitions[0].Competitors[0]
				c2 := event.Competitions[0].Competitors[1]

				home, away := c1, c2
				if c2.HomeAway == "home" {
					home, away = c2, c1
				}

				homeScore, _ := strconv.Atoi(home.Score)
				awayScore, _ := strconv.Atoi(away.Score)

				matches = append(matches, models.Match{
					ID:     event.ID,
					Home:   models.Team{Name: home.Team.Name, Score: homeScore},
					Away:   models.Team{Name: away.Team.Name, Score: awayScore},
					Status: models.Status{
						Reason: struct {
							Short string `json:"short"`
						}{Short: event.Status.Type.Detail},
					},
				})
			}

			// 3. Lock the slice, append our completed league data, and unlock
			mu.Lock()
			allLeagues = append(allLeagues, models.League{
				Name:    leagueName,
				Code:    leagueCode,
				Matches: matches,
			})
			mu.Unlock()

		}(name, code)
	}

	// 4. Pause the main function here until ALL goroutines say they are done
	wg.Wait()
	
	return allLeagues, nil
}

// FetchMatchDetails fetches goal scorers and cards for a specific match
func FetchMatchDetails(leagueCode, matchID string) ([]models.MatchEvent, error) {
	url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/soccer/%s/summary?event=%s", leagueCode, matchID)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var summary espnSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, err
	}

	var events []models.MatchEvent
	for _, ke := range summary.KeyEvents {
		if ke.Type.Text == "Goal" || ke.Type.Text == "Penalty" || ke.Type.Text == "Own Goal" || ke.Type.Text == "Yellow Card" || ke.Type.Text == "Red Card" {
			playerName := "Unknown"
			if len(ke.Participants) > 0 {
				playerName = ke.Participants[0].Athlete.DisplayName
			}
			events = append(events, models.MatchEvent{
				Time:       ke.Clock.DisplayValue,
				PlayerName: playerName,
				TeamName:   ke.Team.Name,
				Type:       ke.Type.Text,
			})
		}
	}
	return events, nil
}