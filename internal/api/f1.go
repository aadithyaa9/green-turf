// internal/api/f1.go
package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aadithyaa9/green-turf/internal/models"
)

type espnF1Scoreboard struct {
	Events []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Date string `json:"date"`
		Status struct {
			Type struct {
				Detail string `json:"detail"`
			} `json:"type"`
		} `json:"status"`
		// NEW: Catch the track details
		Circuit struct {
			FullName string `json:"fullName"`
		} `json:"circuit"`
		Venues []struct {
			FullName string `json:"fullName"`
		} `json:"venues"`
		Competitions []struct {
			Competitors []struct {
				Athlete struct {
					DisplayName string `json:"displayName"`
				} `json:"athlete"`
				Team struct {
					Name string `json:"name"`
				} `json:"team"`
				Status struct {
					DisplayValue string `json:"displayValue"`
				} `json:"status"`
				// NEW: Catch gaps if hidden here
				Linescores []struct {
					DisplayValue string `json:"displayValue"`
				} `json:"linescores"`
				Score string `json:"score"`
				Order int    `json:"order"`
			} `json:"competitors"`
		} `json:"competitions"`
	} `json:"events"`
}

// Helper to reconstruct the 2026 F1 Grid if ESPN's API is missing data!
func inferF1Team(driver string) string {
	d := strings.ToLower(driver)
	if strings.Contains(d, "verstappen") || strings.Contains(d, "perez") || strings.Contains(d, "pérez") || strings.Contains(d, "lawson") { return "Red Bull Racing" }
	if strings.Contains(d, "norris") || strings.Contains(d, "piastri") { return "McLaren" }
	if strings.Contains(d, "leclerc") || strings.Contains(d, "hamilton") { return "Ferrari" }
	if strings.Contains(d, "russell") || strings.Contains(d, "antonelli") { return "Mercedes" }
	if strings.Contains(d, "alonso") || strings.Contains(d, "stroll") || strings.Contains(d, "crawford") { return "Aston Martin" }
	if strings.Contains(d, "gasly") || strings.Contains(d, "doohan") { return "Alpine" }
	if strings.Contains(d, "albon") || strings.Contains(d, "sainz") || strings.Contains(d, "colapinto") { return "Williams" }
	if strings.Contains(d, "ocon") || strings.Contains(d, "bearman") || strings.Contains(d, "magnussen") { return "Haas F1 Team" }
	if strings.Contains(d, "hulkenberg") || strings.Contains(d, "hülkenberg") || strings.Contains(d, "bortoleto") || strings.Contains(d, "bottas") { return "Kick Sauber" }
	if strings.Contains(d, "tsunoda") || strings.Contains(d, "hadjar") || strings.Contains(d, "lindblad") { return "RB" }
	return "Unknown Team"
}
// FetchF1Races gets the current/recent Formula 1 races
func FetchF1Races() ([]models.F1Race, error) {
	url := "https://site.api.espn.com/apis/site/v2/sports/racing/f1/scoreboard"
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var espn espnF1Scoreboard
	if err := json.NewDecoder(resp.Body).Decode(&espn); err != nil {
		return nil, err
	}

	var races []models.F1Race

	for _, event := range espn.Events {
		var drivers []models.F1Driver

		if len(event.Competitions) > 0 {
			for _, comp := range event.Competitions[0].Competitors {
				
				// 1. Get the Time (Check multiple places because ESPN is messy)
				timeStr := comp.Status.DisplayValue
				if timeStr == "" && len(comp.Linescores) > 0 {
					timeStr = comp.Linescores[0].DisplayValue
				}
				if timeStr == "" && comp.Score != "" {
					timeStr = comp.Score + " pts"
				}
				if timeStr == "" {
					timeStr = "-"
				}

				// 2. Get the Team (Fallback to our mathematical mapper if blank)
				teamName := comp.Team.Name
				if teamName == "" {
					teamName = inferF1Team(comp.Athlete.DisplayName)
				}

				drivers = append(drivers, models.F1Driver{
					Position: comp.Order,
					Name:     comp.Athlete.DisplayName,
					Team:     teamName,
					Time:     timeStr,
				})
			}
		}

		sort.Slice(drivers, func(i, j int) bool {
			return drivers[i].Position < drivers[j].Position
		})

		parsedDate, err := time.Parse(time.RFC3339, event.Date)
		formattedDate := event.Date
		if err == nil {
			formattedDate = parsedDate.Local().Format("Mon, Jan 02 2006")
		}

		// 3. Extract the Track Location
		venueName := "Unknown Track"
		if event.Circuit.FullName != "" {
			venueName = event.Circuit.FullName
		} else if len(event.Venues) > 0 {
			venueName = event.Venues[0].FullName
		}

		races = append(races, models.F1Race{
			ID:      event.ID,
			Name:    event.Name,
			Track:   venueName,
			Date:    formattedDate,
			Status:  event.Status.Type.Detail,
			Drivers: drivers,
		})
	}

	return races, nil
}