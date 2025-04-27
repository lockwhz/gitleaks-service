package git

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BranchScanResult struct {
	BranchName string        `json:"branch_name"`
	Success    bool          `json:"success"`
	Findings   []interface{} `json:"findings,omitempty"`
	Error      string        `json:"error,omitempty"`
}

type RepoScanResult struct {
	Repository string             `json:"repository"`
	Results    []BranchScanResult `json:"results"`
}

type Finding struct {
	Description string  `json:"Description"`
	StartLine   int     `json:"StartLine"`
	EndLine     int     `json:"EndLine"`
	StartColumn int     `json:"StartColumn"`
	EndColumn   int     `json:"EndColumn"`
	Match       string  `json:"Match"`
	Secret      string  `json:"Secret"`
	File        string  `json:"File"`
	Commit      string  `json:"Commit"`
	Entropy     float64 `json:"Entropy"`
	Author      string  `json:"Author"`
	Email       string  `json:"Email"`
	Date        string  `json:"Date"`
	Message     string  `json:"Message"`
	Fingerprint string  `json:"Fingerprint"`
	RuleID      string  `json:"RuleID"`
	LeakURL     string  `json:"LeakURL"`
}

// Função pública para ser chamada no main
func ScanRepo(repoURL, accessToken string) (*RepoScanResult, error) {
	baseCloneDir := filepath.Join("./repos")
	os.MkdirAll(baseCloneDir, 0755)

	branches, err := listRemoteBranches(repoURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar branches remotas: %v", err)
	}

	var results []BranchScanResult

	for _, branch := range branches {
		fmt.Printf("🔍 Processando branch: %s\n", branch)
		branchPath := filepath.Join(baseCloneDir, strings.ReplaceAll(branch, "/", "-"))
		os.RemoveAll(branchPath)

		cloneURL := strings.Replace(repoURL, "https://", fmt.Sprintf("https://x-access-token:%s@", accessToken), 1)

		cmd := exec.Command("git", "clone", "--branch", branch, cloneURL, branchPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			results = append(results, BranchScanResult{
				BranchName: branch,
				Success:    false,
				Error:      fmt.Sprintf("Erro ao clonar branch %s: %v", branch, err),
			})
			continue
		}

		entries, _ := os.ReadDir(branchPath)
		fmt.Printf("📂 Arquivos clonados na branch %s:\n", branch)
		for _, entry := range entries {
			fmt.Println(" -", entry.Name())
		}

		result := runGitleaks(branchPath, branch)
		results = append(results, result)
	}

	return &RepoScanResult{
		Repository: repoURL,
		Results:    results,
	}, nil
}

func listRemoteBranches(repoURL string) ([]string, error) {
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("falha ao executar git ls-remote: %v", err)
	}

	var branches []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		if strings.HasPrefix(ref, "refs/heads/") {
			branch := strings.TrimPrefix(ref, "refs/heads/")
			branches = append(branches, branch)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler a saída do ls-remote: %v", err)
	}

	return branches, nil
}

func runGitleaks(repoPath, branchName string) BranchScanResult {
	gitleaksPath := "repos/gitleaks"

	info, err := os.Stat(gitleaksPath)
	if err != nil || info.Mode()&0111 == 0 {
		return BranchScanResult{
			BranchName: branchName,
			Success:    false,
			Error:      "❌ gitleaks não encontrado ou sem permissão de execução",
		}
	}

	tempOutput := filepath.Join(repoPath, "gitleaks_output.json")
	cmd := exec.Command(gitleaksPath, "git",
		"--report-path", tempOutput,
		"--report-format", "json",
		"--no-banner",
		repoPath)

	cmd.Env = append(os.Environ(), "GITLEAKS_LOG_LEVEL=FATAL")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := BranchScanResult{
		BranchName: branchName,
		Success:    true,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 1:
				fmt.Printf("🟡 Segredos encontrados na branch %s\n", branchName)
			default:
				result.Success = false
				result.Error = fmt.Sprintf("❌ Falha Gitleaks (%d): %s", exitErr.ExitCode(), stderr.String())
				return result
			}
		}
	}

	reportContent, err := os.ReadFile(tempOutput)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("❌ Falha ao ler o relatório do Gitleaks: %v", err)
		return result
	}

	if len(reportContent) == 0 {
		result.Error = "🟡 Gitleaks executado, mas sem saída JSON."
		return result
	}

	var findings []Finding
	if err := json.Unmarshal(reportContent, &findings); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("❌ Erro ao decodificar JSON da saída do Gitleaks: %v\nSaída: %s", err, string(reportContent))
		return result
	}

	for _, finding := range findings {
		// Aqui você pode fazer algo (ex: salvar no banco)
		fmt.Printf("\n\033[1mFinding:\033[0m %+v\n", finding)
	}

	// Convertendo para []interface{} pra manter compatibilidade
	result.Findings = make([]interface{}, len(findings))
	for i, f := range findings {
		result.Findings[i] = f
	}

	return result
}
