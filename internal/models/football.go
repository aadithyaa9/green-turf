package models

// FotmobResponse is kept just in case you switch back, but we rely on ESPN now
type FotmobResponse struct {
	Leagues []League `json:"leagues"`
}

type League struct {
	ID      int     `json:"primaryId"`
	Name    string  `json:"name"`
	Code    string  // NEW: We need this to fetch the summary (e.g., "eng.1")
	Matches []Match `json:"matches"`
}

type Team struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type Match struct {
	ID     string       // NEW: ESPN's unique match ID
	Home   Team         `json:"home"`
	Away   Team         `json:"away"`
	Status Status       `json:"status"`
	Events []MatchEvent // NEW: To hold goals and cards
}

type Status struct {
	Finished bool   `json:"finished"`
	Started  bool   `json:"started"`
	ScoreStr string `json:"scoreStr"`
	Reason   struct {
		Short string `json:"short"`
	} `json:"reason"`
}

// NEW: Struct for match timeline events
type MatchEvent struct {
	Time       string
	PlayerName string
	TeamName   string
	Type       string // e.g., "Goal", "Yellow Card", "Red Card"
}