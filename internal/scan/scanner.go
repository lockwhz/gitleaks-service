package scan

import "yourproject/models"

// Scanner define uma interface para executar o scanner.
type Scanner interface {
	Run(repoPath string) ([]models.GitleaksFinding, error)
}
