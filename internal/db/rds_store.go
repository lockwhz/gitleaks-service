package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"yourproject/models"
	"yourproject/internal/logger"
)

// RDSStore implementa DataStore usando um banco PostgreSQL (RDS).
type RDSStore struct {
	DB *sql.DB
}

func (r *RDSStore) UpdateScanStatus(scanID, status string) error {
	start := time.Now()
	defer logger.Trace("UpdateScanStatus", start)

	query := `UPDATE scans SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.DB.ExecContext(context.Background(), query, status, time.Now(), scanID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status do scan %s: %v", scanID, err)
	}
	return nil
}

func (r *RDSStore) InsertFinding(job *models.ScanJob, finding models.GitleaksFinding) error {
	start := time.Now()
	defer logger.Trace("InsertFinding", start)

	query := `
		INSERT INTO resultado_exploracao_credencial_exposta (
			codigo_resultado_exploracao,
			codigo_exploracao_credencial_exposta,
			codigo_repositorio,
			nome_divisao_repositorio,
			nome_caminho_arquivo,
			numero_linha_inicio,
			nome_regra_credencial,
			nome_unico_credencial,
			data_hora_criacao_registro
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	resID := uuid.New().String()
	expID := uuid.New().String()

	_, err := r.DB.ExecContext(context.Background(), query,
		resID,
		expID,
		job.RepositoryID,
		job.Sigla,
		finding.File,
		finding.StartLine,
		finding.RuleID,
		finding.Secret,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("erro ao inserir achado: %v", err)
	}
	return nil
}
