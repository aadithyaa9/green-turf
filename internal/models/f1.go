// internal/models/f1.go
package models

type F1Race struct {
	ID      string
	Name    string
	Track   string // NEW: Circuit location
	Date    string
	Status  string
	Drivers []F1Driver
}

type F1Driver struct {
	Position int
	Name     string
	Team     string
	Time     string
}