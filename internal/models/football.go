package models

type FotmobResponse struct {
	Leagues []League `json:"leagues"`
}

type League struct {
	ID      int     `json:"primaryId"`
	Name    string  `json:"name"`
	Matches []Match `json:"matches"`
}

type Team struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type Match struct {
	ID     int    `json:"id"`
	Home   Team   `json:"home"`
	Away   Team   `json:"away"`
	Status Status `json:"status"`
}


type Status struct {
	Finished bool   `json:"finished"`
	Started  bool   `json:"started"`
	ScoreStr string `json:"scoreStr"`
	Reason   struct {
		Short string `json:"short"` // e.g., "FT", "HT", "75'" , remember you idiot !
	} `json:"reason"`
}