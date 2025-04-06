package scan

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"yourproject/models"
	"yourproject/internal/logger"
)

type GitleaksScanner struct {
	GitleaksPath string
}

func (s *GitleaksScanner) Run(repoPath string) ([]models.GitleaksFinding, error) {
	start := time.Now()
	defer logger.Trace("RunGitleaks", start)

	tempFile, err := os.CreateTemp("", "gitleaks_report_*.json")
	if err != nil {
		return nil, fmt.Errorf("erro ao criar arquivo temporário: %v", err)
	}
	reportPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(reportPath)

	cmd := exec.Command(s.GitleaksPath,
		"detect",
		"--source="+repoPath,
		"--report-format=json",
		"--report-path="+reportPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gitleaks detect falhou: %v, output: %s", err, string(output))
	}

	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o relatório: %v", err)
	}

	var findings []models.GitleaksFinding
	if err := json.Unmarshal(reportContent, &findings); err != nil {
		return nil, fmt.Errorf("erro ao parsear JSON do gitleaks: %v", err)
	}
	return findings, nil
}
