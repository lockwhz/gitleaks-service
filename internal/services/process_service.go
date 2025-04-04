package services

import (
	"database/sql"
	"fmt"
	"time"

	"yourproject/models"
	"yourproject/internal/db"
	"yourproject/internal/git"
	"yourproject/internal/logger"
	"yourproject/internal/scan"
)

// ProcessJob executa o fluxo completo de um job.
// Ele atualiza o status, clona o repositório (se habilitado), executa o scanner (se habilitado) e insere os achados.
func ProcessJob(job *models.ScanJob, dbConn *sql.DB, gitClient git.GitClient, scanner scan.Scanner, cloneSem chan struct{}) error {
	start := time.Now()
	logger.Log.Debugf("ProcessService: Iniciando processamento do job %s", job.ScanID)

	// Atualiza o status para "running".
	if err := db.UpdateScanStatus(dbConn, job.ScanID, "running"); err != nil {
		logger.Log.Errorf("ProcessService: Erro ao atualizar status do scan %s: %v", job.ScanID, err)
		return err
	}

	// Monta a URL de clonagem.
	repoURL := fmt.Sprintf("https://github.com/%s", job.RepositoryFullName)

	var repoPath string
	if EnableClone() {
		cloneSem <- struct{}{}
		var err error
		repoPath, err = gitClient.CloneRepo(repoURL)
		<-cloneSem
		if err != nil {
			db.UpdateScanStatus(dbConn, job.ScanID, "error")
			return fmt.Errorf("ProcessService: erro ao clonar repositório: %v", err)
		}
		logger.Log.Debugf("ProcessService: Repositório clonado em %s", repoPath)
	} else {
		logger.Log.Debug("ProcessService: Clone desabilitado; pulando etapa de clone")
		repoPath = ""
	}

	var findings []models.GitleaksFinding
	if EnableScan() {
		f, err := scanner.Run(repoPath)
		if err != nil {
			db.UpdateScanStatus(dbConn, job.ScanID, "error")
			return fmt.Errorf("ProcessService: erro ao executar o scanner: %v", err)
		}
		findings = f
		logger.Log.Debugf("ProcessService: Scanner encontrou %d achados para o job %s", len(findings), job.ScanID)
	} else {
		logger.Log.Debug("ProcessService: Scanner desabilitado; retornando findings vazios")
		findings = []models.GitleaksFinding{}
	}

	for _, f := range findings {
		if err := db.InsertFinding(dbConn, job, f); err != nil {
			logger.Log.Errorf("ProcessService: erro ao inserir achado para o job %s: %v", job.ScanID, err)
		}
	}

	if err := db.UpdateScanStatus(dbConn, job.ScanID, "success"); err != nil {
		logger.Log.Errorf("ProcessService: erro ao atualizar status final do scan %s: %v", job.ScanID, err)
	}

	logger.Log.Debugf("ProcessService: Processamento do job %s concluído em %d ms", job.ScanID, time.Since(start).Milliseconds())
	return nil
}

// EnableClone verifica se a funcionalidade de clone está habilitada.
func EnableClone() bool {
	return GetEnvAsBool("ENABLE_GIT_CLONE", true)
}

// EnableScan verifica se o scanner está habilitado.
func EnableScan() bool {
	return GetEnvAsBool("ENABLE_GITLEAKS", true)
}

// GetEnvAsBool retorna o valor booleano de uma variável de ambiente, com um padrão.
func GetEnvAsBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}
