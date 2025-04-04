package db

import "yourproject/models"

// DataStore define a interface para operações de banco de dados.
type DataStore interface {
	UpdateScanStatus(scanID, status string) error
	InsertFinding(job *models.ScanJob, finding models.GitleaksFinding) error
}
