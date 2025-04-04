package models

import "time"

type ScanJob struct {
	ScanID             string    `json:"scan_id"`
	RepositoryID       string    `json:"repository_id"`
	RepositoryFullName string    `json:"repository_full_name"` // Ex.: "org/repo"
	RepositorySize     int       `json:"repository_size"`
	RepositoryLanguage string    `json:"repository_language"`
	Sigla              string    `json:"sigla"`
	MessageCreatedAt   time.Time `json:"message_created_at"`
}

type GitleaksFinding struct {
	Description string   `json:"Description"`
	File        string   `json:"File"`
	StartLine   int      `json:"StartLine"`
	RuleID      string   `json:"RuleID"`
	Secret      string   `json:"Secret"`
	Tags        []string `json:"Tags"`
}
