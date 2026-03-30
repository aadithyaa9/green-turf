// internal/tui/football.go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/aadithyaa9/green-turf/internal/api"
	"github.com/aadithyaa9/green-turf/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- STYLES ---
var (
	// Define our team colors (Cyan for Home, Pink for Away)
	homeColor = lipgloss.Color("86")  // Aqua/Cyan
	awayColor = lipgloss.Color("212") // Pink

	// Text styles
	homeStyle = lipgloss.NewStyle().Foreground(homeColor).Bold(true)
	awayStyle = lipgloss.NewStyle().Foreground(awayColor).Bold(true)
	
	// Scoreboard style (Dark grey background, white text)
	scoreStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)

	// Box styles for the side-by-side event columns
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Width(35) // Fixed width so the columns look even
)

type dataMsg []models.League
type detailsMsg []models.MatchEvent
type errMsg error
type tickMsg time.Time

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

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*60, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

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
		if m.selectedLeague != nil {
			for i := range m.leagues {
				if m.leagues[i].Code == m.selectedLeague.Code {
					m.selectedLeague = &m.leagues[i]
					if m.selectedMatch != nil {
						for j := range m.selectedLeague.Matches {
							if m.selectedLeague.Matches[j].ID == m.selectedMatch.ID {
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

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, fetchFootballData)
		if m.selectedMatch != nil {
			cmds = append(cmds, fetchMatchDetailsCmd(m.selectedLeague.Code, m.selectedMatch.ID))
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)
	}
	
	return m, nil
}

func (m FootballModel) View() string {
	if m.err != nil { return fmt.Sprintf("\n❌ Error: %v\n", m.err) }
	if m.isLoading { return "\n⏳ Fetching live match data...\n" }

	var b strings.Builder

	// LEVEL 3: The Beautiful Match Dashboard
	if m.selectedMatch != nil {
		homeName := m.selectedMatch.Home.Name
		awayName := m.selectedMatch.Away.Name
		scoreTxt := fmt.Sprintf(" %d - %d ", m.selectedMatch.Home.Score, m.selectedMatch.Away.Score)

		// 1. Render the top scoreboard banner
		header := fmt.Sprintf("🏟️  %s %s %s\n", 
			homeStyle.Render(homeName), 
			scoreStyle.Render(scoreTxt), 
			awayStyle.Render(awayName),
		)
		b.WriteString(header)
		b.WriteString(fmt.Sprintf("   Status: %s\n\n", m.selectedMatch.Status.Reason.Short))

		// 2. Render the events
		if m.isFetchingDetails {
			b.WriteString("   ⏳ Loading match events...\n")
		} else if len(m.selectedMatch.Events) == 0 {
			b.WriteString("   No major events yet (or data unavailable).\n")
		} else {
			var homeEvents, awayEvents strings.Builder

			// Sort events into Home and Away buckets using Team ID!
			for _, event := range m.selectedMatch.Events {
				icon := "⚽"
				if event.Type == "Yellow Card" { icon = "🟨" }
				if event.Type == "Red Card" { icon = "🟥" }
				if event.Type == "Own Goal" { icon = "🤦" }

				eventLine := fmt.Sprintf("%s [%s'] %s\n", icon, event.Time, event.PlayerName)

				// Use exact Team ID matching
				if event.TeamID == m.selectedMatch.Home.ID {
					homeEvents.WriteString(eventLine)
				} else if event.TeamID == m.selectedMatch.Away.ID {
					awayEvents.WriteString(eventLine)
				} else {
					// Fallback just in case ID is missing from API
					if strings.Contains(strings.ToLower(m.selectedMatch.Home.Name), strings.ToLower(event.TeamName)) {
						homeEvents.WriteString(eventLine)
					} else {
						awayEvents.WriteString(eventLine)
					}
				}
			}

			// Render the colored boxes for each team
			homeBox := boxStyle.Copy().BorderForeground(homeColor).Render(
				homeStyle.Render(homeName+" Events") + "\n\n" + homeEvents.String(),
			)
			awayBox := boxStyle.Copy().BorderForeground(awayColor).Render(
				awayStyle.Render(awayName+" Events") + "\n\n" + awayEvents.String(),
			)

			// Join them side-by-side
			dashboard := lipgloss.JoinHorizontal(lipgloss.Top, homeBox, "   ", awayBox)
			b.WriteString(dashboard + "\n")
		}
		b.WriteString("\n[Press 'b' to go back, 'q' to quit]\n")
		return b.String()
	}

	// LEVEL 2
	if m.selectedLeague != nil {
		b.WriteString(fmt.Sprintf("🏆 %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.selectedLeague.Name)))
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