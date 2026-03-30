package models

type FotmobResponse struct {
	Leagues []League `json:"leagues"`
}

type League struct {
	ID      int     `json:"primaryId"`
	Name    string  `json:"name"`
	Code    string  
	Matches []Match `json:"matches"`
}

type Team struct {
	ID    string 
	Name  string
	Score int   
}

type Match struct {
	ID     string       
	Date   string       // NEW: To hold the formatted date
	Home   Team         
	Away   Team         
	Status Status       
	Events []MatchEvent 
}

type Status struct {
	Finished bool   `json:"finished"`
	Started  bool   `json:"started"`
	ScoreStr string `json:"scoreStr"`
	Reason   struct {
		Short string `json:"short"`
	} `json:"reason"`
}

type MatchEvent struct {
	Time       string
	PlayerName string
	TeamID     string 
	TeamName   string
	Type       string
}