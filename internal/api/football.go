// internal/api/football.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
	"strings"

	"github.com/aadithyaa9/green-turf/internal/models"
)

type espnScoreboard struct {
	Events []struct {
		ID           string `json:"id"`
		Date         string `json:"date"` // NEW: Capture the ESPN timestamp
		Competitions []struct {
			Competitors []struct {
				HomeAway string `json:"homeAway"`
				Score    string `json:"score"`
				Team     struct {
					ID   string `json:"id"`
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
			ID   string `json:"id"` // CAPTURE TEAM ID
			Name string `json:"name"`
		} `json:"team"`
		Participants []struct {
			Athlete struct {
				DisplayName string `json:"displayName"`
			} `json:"athlete"`
		} `json:"participants"`
	} `json:"keyEvents"`
}

// FetchFootballMatches concurrently gets scores from top European leagues
func FetchFootballMatches(date string) ([]models.League, error) {
	now := time.Now()
	// Removed the unused 'today' variable
	yesterday := now.AddDate(0, 0, -14).Format("20060102")
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

			url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/soccer/%s/scoreboard?dates=%s-%s", 
				leagueCode, yesterday, endDate)
			
			resp, err := client.Get(url)
			if err != nil { return }
			defer resp.Body.Close()

			var espn espnScoreboard
			if err := json.NewDecoder(resp.Body).Decode(&espn); err != nil { return }

			var matches []models.Match
			for _, event := range espn.Events {
				if len(event.Competitions) == 0 || len(event.Competitions[0].Competitors) < 2 { continue }

				// NEW: Parse the ESPN date (RFC3339 format) into a readable string
				parsedDate, err := time.Parse(time.RFC3339, event.Date)
				formattedDate := event.Date // fallback to raw string
				if err == nil {
					// Convert to local time and format like: "Sat, Mar 30 2024"
					formattedDate = parsedDate.Local().Format("Mon, Jan 02 2006")
				}

				c1 := event.Competitions[0].Competitors[0]
				c2 := event.Competitions[0].Competitors[1]
				home, away := c1, c2
				if c2.HomeAway == "home" { home, away = c2, c1 }

				homeScore, _ := strconv.Atoi(home.Score)
				awayScore, _ := strconv.Atoi(away.Score)

				matches = append(matches, models.Match{
					ID:   event.ID,
					Date: formattedDate, // NEW: Save the parsed date
					Home: models.Team{ID: home.Team.ID, Name: home.Team.Name, Score: homeScore},
					Away: models.Team{ID: away.Team.ID, Name: away.Team.Name, Score: awayScore},
					Status: models.Status{
						Reason: struct {
							Short string `json:"short"`
						}{Short: event.Status.Type.Detail},
					},
				})
			}

			mu.Lock()
			allLeagues = append(allLeagues, models.League{ Name: leagueName, Code: leagueCode, Matches: matches })
			mu.Unlock()
		}(name, code)
	}

	wg.Wait()
	return allLeagues, nil
}

// FetchMatchDetails hits the 'summary' endpoint to get goal scorers
// FetchMatchDetails hits the 'summary' endpoint to get goal scorers
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
		typeText := strings.ToLower(ke.Type.Text)
		
		// The Fix: Safe substring checking to avoid "scoRED" triggering a Red Card!
		isOwnGoal := strings.Contains(typeText, "own")
		isPenalty := strings.Contains(typeText, "penalty")
		isGoal := strings.Contains(typeText, "goal") || isPenalty || strings.Contains(typeText, "scored")
		
		isYellow := strings.Contains(typeText, "yellow")
		isRed := strings.Contains(typeText, "red") && !strings.Contains(typeText, "scored")
		
		isCard := strings.Contains(typeText, "card") || isYellow || isRed

		if isGoal || isCard {
			playerName := "Unknown"
			if len(ke.Participants) > 0 {
				playerName = ke.Participants[0].Athlete.DisplayName
			}
			
			cleanType := "Goal"
			if isOwnGoal {
				cleanType = "Own Goal"
			} else if isRed {
				cleanType = "Red Card"
			} else if isYellow {
				cleanType = "Yellow Card"
			}

			cleanTime := strings.ReplaceAll(ke.Clock.DisplayValue, "'", "")

			events = append(events, models.MatchEvent{
				Time:       cleanTime,
				PlayerName: playerName,
				TeamID:     ke.Team.ID,
				TeamName:   ke.Team.Name,
				Type:       cleanType,
			})
		}
	}
	return events, nil
}