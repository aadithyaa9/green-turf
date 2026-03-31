// internal/tui/f1.go
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aadithyaa9/green-turf/internal/api"
	"github.com/aadithyaa9/green-turf/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	f1BrandColor = lipgloss.Color("196")
	f1TitleStyle = lipgloss.NewStyle().Foreground(f1BrandColor).Bold(true).MarginBottom(1)
	
	posStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Width(3)
	driverBaseStyle = lipgloss.NewStyle().Bold(true).Width(16)
	timeStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(10)
)

func getTeamStyle(teamName string) lipgloss.Style {
	base := lipgloss.NewStyle()
	lowerName := strings.ToLower(teamName)

	if strings.Contains(lowerName, "ferrari") { return base.Foreground(lipgloss.Color("#EF1A2D")) }
	if strings.Contains(lowerName, "red bull") { return base.Foreground(lipgloss.Color("#3671C6")) }
	if strings.Contains(lowerName, "mercedes") { return base.Foreground(lipgloss.Color("#27F4D2")) }
	if strings.Contains(lowerName, "mclaren") { return base.Foreground(lipgloss.Color("#FF8000")) }
	if strings.Contains(lowerName, "aston martin") { return base.Foreground(lipgloss.Color("#229971")) }
	if strings.Contains(lowerName, "alpine") { return base.Foreground(lipgloss.Color("#FF87BC")) }
	if strings.Contains(lowerName, "williams") { return base.Foreground(lipgloss.Color("#64C4FF")) }
	if strings.Contains(lowerName, "haas") { return base.Foreground(lipgloss.Color("#FFFFFF")) }
	if strings.Contains(lowerName, "kick") || strings.Contains(lowerName, "sauber") { return base.Foreground(lipgloss.Color("#52E252")) }
	if strings.Contains(lowerName, "rb") || strings.Contains(lowerName, "alphatauri") { return base.Foreground(lipgloss.Color("#6692FF")) }

	return base.Foreground(lipgloss.Color("246"))
}

// THE ENGINE: Parses ESPN's chaotic gap strings into workable math
func parseGapToSeconds(gap string) float64 {
	gap = strings.ToLower(strings.TrimSpace(gap))
	if gap == "" || gap == "-" || gap == "leader" { return 0.0 }
	if strings.Contains(gap, "lap") || strings.Contains(gap, "l") { return 100.0 } // Push lapped cars far back
	if strings.Contains(gap, "out") || strings.Contains(gap, "dnf") || strings.Contains(gap, "dns") { return 120.0 } // Push DNFs to the absolute end

	gap = strings.ReplaceAll(gap, "+", "")
	gap = strings.ReplaceAll(gap, "s", "")

	// Handle minute gaps (e.g., "1:23.4")
	if strings.Contains(gap, ":") {
		parts := strings.Split(gap, ":")
		if len(parts) == 2 {
			m, _ := strconv.ParseFloat(parts[0], 64)
			s, _ := strconv.ParseFloat(parts[1], 64)
			return (m * 60) + s
		}
	}

	// Normal seconds
	val, err := strconv.ParseFloat(gap, 64)
	if err != nil { return 80.0 } // Unknown fallback
	return val
}

type f1DataMsg []models.F1Race

type F1Model struct {
	races        []models.F1Race
	cursor       int
	selectedRace *models.F1Race
	isLoading    bool
	err          error
}

func NewF1Model() F1Model { return F1Model{isLoading: true} }

func fetchF1Data() tea.Msg {
	races, err := api.FetchF1Races()
	if err != nil { return errMsg(err) }
	return f1DataMsg(races)
}

func f1TickCmd() tea.Cmd {
	return tea.Tick(time.Second*60, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m F1Model) Init() tea.Cmd { return tea.Batch(fetchF1Data, f1TickCmd()) }

func (m F1Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q": return m, tea.Quit
		case "up", "k":
			if m.selectedRace == nil && m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.selectedRace == nil && m.cursor < len(m.races)-1 { m.cursor++ }
		case "enter":
			if m.selectedRace == nil && len(m.races) > 0 { m.selectedRace = &m.races[m.cursor] }
		case "b", "esc":
			if m.selectedRace != nil { m.selectedRace = nil }
		}

	case f1DataMsg:
		m.races = msg
		m.isLoading = false
		if m.selectedRace != nil {
			for i := range m.races {
				if m.races[i].ID == m.selectedRace.ID { m.selectedRace = &m.races[i] }
			}
		}
		return m, nil

	case errMsg:
		m.err = msg
		m.isLoading = false
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchF1Data, f1TickCmd())
	}
	return m, nil
}

func (m F1Model) View() string {
	if m.err != nil { return fmt.Sprintf("\n❌ Error: %v\n", m.err) }
	if m.isLoading { return "\n🏎️  Fetching live F1 telemetry...\n" }

	var b strings.Builder

	// LEVEL 2: The 1D Track Map
	if m.selectedRace != nil {
		b.WriteString(f1TitleStyle.Render(fmt.Sprintf("🏎️  %s", m.selectedRace.Name)) + "\n")
		b.WriteString(fmt.Sprintf("📍 %s\n", m.selectedRace.Track))
		b.WriteString(fmt.Sprintf("📅 %s  |  Status: %s\n\n", m.selectedRace.Date, m.selectedRace.Status))

		if len(m.selectedRace.Drivers) == 0 {
			b.WriteString("Grid data not available yet.\n")
		} else {
			// 1. Calculate the scaling factor based on the leader and the backmarker
			maxGap := 5.0 // Minimum 5 seconds to prevent erratic jumping early on
			gaps := make([]float64, len(m.selectedRace.Drivers))
			
			for i, d := range m.selectedRace.Drivers {
				g := parseGapToSeconds(d.Time)
				gaps[i] = g
				// Find the highest gap that isn't a Lapped/OUT car to scale the track
				if g > maxGap && g < 100.0 { 
					maxGap = g
				}
			}

			// 2. Draw the Grid
			trackWidth := 40 // The physical width of the ASCII track
			b.WriteString(fmt.Sprintf("%s %s %s  %s\n%s\n", 
				posStyle.Render("P"), 
				driverBaseStyle.Copy().Foreground(lipgloss.Color("241")).Render("DRIVER"), 
				lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(13).Render("CONSTRUCTOR"), 
				lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("RELATIVE TRACK POSITION (Leader to Backmarker)"),
				strings.Repeat("─", 85),
			))

			for i, d := range m.selectedRace.Drivers {
				teamColorStyle := getTeamStyle(d.Team)
				driverStyling := driverBaseStyle.Copy().Inherit(teamColorStyle)

				// Calculate exact position on the 40-character line
				gapSecs := gaps[i]
				pos := int((gapSecs / maxGap) * float64(trackWidth))
				
				// Clamp values so lapped cars don't break the terminal width
				if pos > trackWidth { pos = trackWidth }
				if pos < 0 { pos = 0 }

				// Draw the track: "----🏎️----------------"
				trackLine := strings.Repeat("┈", pos) + "🏎️" + strings.Repeat("┈", trackWidth - pos)

				b.WriteString(fmt.Sprintf("%s %s %s  %s  %s\n",
					posStyle.Render(fmt.Sprintf("%d", d.Position)),
					driverStyling.Render(d.Name),
					teamColorStyle.Width(13).Render(d.Team),
					teamColorStyle.Render(trackLine), // Color the track to match the team!
					timeStyle.Render(d.Time),
				))
			}
		}
		
		b.WriteString("\n[Press 'b' to go back, 'q' to quit]\n")
		return b.String()
	}

	// LEVEL 1: Select a Race
	b.WriteString(f1TitleStyle.Render("🏎️  Formula 1 Telemetry Map (Auto-updates every 60s)") + "\n\n")
	if len(m.races) == 0 { return "No F1 events currently found.\n\n[Press 'q' to quit]" }

	for i, race := range m.races {
		cursor := "  "
		if m.cursor == i { cursor = ">>" }
		b.WriteString(fmt.Sprintf("%s %s (%s)\n", cursor, race.Name, race.Status))
	}
	b.WriteString("\n[Use Up/Down to move, Enter to select, 'q' to quit]\n")
	return b.String()
}