// internal/tui/football.go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/aadithyaa9/green-turf/internal/api"
	"github.com/aadithyaa9/green-turf/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

type dataMsg []models.League
type detailsMsg []models.MatchEvent
type errMsg error
type tickMsg time.Time // NEW: The alarm clock message

type FootballModel struct {
	leagues           []models.League
	leagueCursor      int
	selectedLeague    *models.League
	
	matchCursor       int
	selectedMatch     *models.Match
	
	isLoading         bool
	isFetchingDetails bool
	err               error
}

func NewFootballModel() FootballModel {
	return FootballModel{isLoading: true}
}

func fetchFootballData() tea.Msg {
	// We use the same wide date range here to match the API
	today := time.Now().Format("20060102")
	leagues, err := api.FetchFootballMatches(today)
	if err != nil { return errMsg(err) }
	return dataMsg(leagues)
}

func fetchMatchDetailsCmd(leagueCode, matchID string) tea.Cmd {
	return func() tea.Msg {
		events, err := api.FetchMatchDetails(leagueCode, matchID)
		if err != nil { return errMsg(err) }
		return detailsMsg(events)
	}
}

// NEW: The background timer (fires every 60 seconds)
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*60, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// UPDATE: Start both the initial fetch AND the timer simultaneously
func (m FootballModel) Init() tea.Cmd {
	return tea.Batch(fetchFootballData, tickCmd())
}

func (m FootballModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		
		case "up", "k":
			if m.selectedLeague == nil && m.leagueCursor > 0 {
				m.leagueCursor--
			} else if m.selectedLeague != nil && m.selectedMatch == nil && m.matchCursor > 0 {
				m.matchCursor--
			}
			
		case "down", "j":
			if m.selectedLeague == nil && m.leagueCursor < len(m.leagues)-1 {
				m.leagueCursor++
			} else if m.selectedLeague != nil && m.selectedMatch == nil && m.matchCursor < len(m.selectedLeague.Matches)-1 {
				m.matchCursor++
			}
			
		case "enter":
			if m.selectedLeague == nil && len(m.leagues) > 0 {
				m.selectedLeague = &m.leagues[m.leagueCursor]
				m.matchCursor = 0
			} else if m.selectedLeague != nil && m.selectedMatch == nil && len(m.selectedLeague.Matches) > 0 {
				m.selectedMatch = &m.selectedLeague.Matches[m.matchCursor]
				m.isFetchingDetails = true
				return m, fetchMatchDetailsCmd(m.selectedLeague.Code, m.selectedMatch.ID)
			}
			
		case "b", "esc":
			if m.selectedMatch != nil {
				m.selectedMatch = nil 
			} else if m.selectedLeague != nil {
				m.selectedLeague = nil
			}
		}

	case dataMsg:
		m.leagues = msg
		m.isLoading = false
		
		// THE POINTER RE-BINDING LOGIC:
		// If new data arrives while the user is deep in a menu, we must update 
		// their pointers to point at the fresh data, otherwise the UI breaks!
		if m.selectedLeague != nil {
			for i := range m.leagues {
				if m.leagues[i].Code == m.selectedLeague.Code {
					m.selectedLeague = &m.leagues[i]
					
					// Re-bind the selected match if they are viewing details
					if m.selectedMatch != nil {
						for j := range m.selectedLeague.Matches {
							if m.selectedLeague.Matches[j].ID == m.selectedMatch.ID {
								// Preserve the events we already fetched
								savedEvents := m.selectedMatch.Events
								m.selectedMatch = &m.selectedLeague.Matches[j]
								m.selectedMatch.Events = savedEvents
							}
						}
					}
				}
			}
		}
		return m, nil

	case detailsMsg:
		if m.selectedMatch != nil {
			m.selectedMatch.Events = msg
			m.isFetchingDetails = false
		}
		return m, nil

	case errMsg:
		m.err = msg
		m.isLoading = false
		m.isFetchingDetails = false
		return m, nil

	// NEW: When the 60-second alarm rings
	case tickMsg:
		var cmds []tea.Cmd
		
		// 1. Fetch the main scoreboard again
		cmds = append(cmds, fetchFootballData)
		
		// 2. If they are actively watching a match, fetch the latest goals for it too!
		if m.selectedMatch != nil {
			cmds = append(cmds, fetchMatchDetailsCmd(m.selectedLeague.Code, m.selectedMatch.ID))
		}
		
		// 3. Reset the alarm clock for the next 60 seconds
		cmds = append(cmds, tickCmd())
		
		return m, tea.Batch(cmds...)
	}
	
	return m, nil
}

func (m FootballModel) View() string {
	if m.err != nil { return fmt.Sprintf("\n❌ Error: %v\n", m.err) }
	if m.isLoading { return "\n⏳ Fetching live match data...\n" }

	var b strings.Builder

	// LEVEL 3
	if m.selectedMatch != nil {
		b.WriteString(fmt.Sprintf("🏟️  %s %d - %d %s\n", 
			m.selectedMatch.Home.Name, m.selectedMatch.Home.Score, 
			m.selectedMatch.Away.Score, m.selectedMatch.Away.Name))
		b.WriteString(fmt.Sprintf("Status: %s\n\n", m.selectedMatch.Status.Reason.Short))

		if m.isFetchingDetails {
			b.WriteString("⏳ Loading match events...\n")
		} else {
			if len(m.selectedMatch.Events) == 0 {
				b.WriteString("No major events yet (or data unavailable).\n")
			} else {
				b.WriteString("⏱️  Match Events:\n")
				for _, event := range m.selectedMatch.Events {
					icon := "⚽"
					if event.Type == "Yellow Card" { icon = "🟨" }
					if event.Type == "Red Card" { icon = "🟥" }
					if event.Type == "Own Goal" { icon = "🤦" }

					b.WriteString(fmt.Sprintf("  %s [%s'] %s (%s)\n", icon, event.Time, event.PlayerName, event.TeamName))
				}
			}
		}
		b.WriteString("\n[Press 'b' to go back, 'q' to quit]\n")
		return b.String()
	}

	// LEVEL 2
	if m.selectedLeague != nil {
		b.WriteString(fmt.Sprintf("🏆 %s\n\n", m.selectedLeague.Name))
		if len(m.selectedLeague.Matches) == 0 {
			b.WriteString("No matches found.\n")
		} else {
			for i, match := range m.selectedLeague.Matches {
				cursor := "  "
				if m.matchCursor == i { cursor = ">>" } 
				
				b.WriteString(fmt.Sprintf("%s %s %d - %d %s (%s)\n", 
					cursor, match.Home.Name, match.Home.Score, match.Away.Score, match.Away.Name, match.Status.Reason.Short))
			}
		}
		b.WriteString("\n[Use Up/Down to move, Enter to view details, 'b' to go back]\n")
		return b.String()
	}

	// LEVEL 1
	b.WriteString("⚽ Select a League (Auto-updates every 60s):\n\n")
	for i, league := range m.leagues {
		cursor := "  "
		if m.leagueCursor == i { cursor = ">>" }
		b.WriteString(fmt.Sprintf("%s %s\n", cursor, league.Name))
	}
	b.WriteString("\n[Use Up/Down to move, Enter to select, 'q' to quit]\n")
	return b.String()
}