package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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

func main() {
	repoURL := "https://github.com/lockwhz/gitleaks-service.git"
	accessToken := "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"

	result, err := scanRepo(repoURL, accessToken)
	if err != nil {
		log.Fatalf("Falha no scan: %v", err)
	}

	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Erro ao gerar JSON: %v", err)
	}
	fmt.Println(string(jsonOutput))
}

func scanRepo(repoURL, accessToken string) (*RepoScanResult, error) {
	baseCloneDir := filepath.Join("./repos")
	os.MkdirAll(baseCloneDir, 0755)

	branches := []string{"main", "develop", "feature/test"}

	var results []BranchScanResult

	for _, branch := range branches {
		fmt.Printf("🔍 Processando branch: %s\n", branch)
		branchPath := filepath.Join(baseCloneDir, strings.ReplaceAll(branch, "/", "-"))
		os.RemoveAll(branchPath)

		cloneURL := strings.Replace(repoURL, "https://", fmt.Sprintf("https://x-access-token:%s@", accessToken), 1)

		cmd := exec.Command("git", "clone", "--branch", branch, "--no-single-branch", cloneURL, branchPath)
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

	var findings []map[string]interface{}
	if err := json.Unmarshal(reportContent, &findings); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("❌ Erro ao decodificar JSON da saída do Gitleaks: %v\nSaída: %s", err, string(reportContent))
		return result
	}

	result.Findings = make([]interface{}, len(findings))
	for i, f := range findings {
		result.Findings[i] = f
		fmt.Println("\n\033[1mFinding Detalhado:\033[0m")
		fmt.Printf("Line:        %v\n", f["Line"])
		fmt.Printf("Secret:      \033[38;5;208m%v\033[0m\n", f["Secret"])
		fmt.Printf("RuleID:      %v\n", f["RuleID"])
		fmt.Printf("Entropy:     %v\n", f["Entropy"])
		fmt.Printf("File:        %v\n", f["File"])
		fmt.Printf("Line:        %v\n", f["StartLine"])
		fmt.Printf("Commit:      %v\n", f["Commit"])
		fmt.Printf("Author:      %v\n", f["Author"])
		fmt.Printf("Email:       %v\n", f["Email"])
		fmt.Printf("Date:        %v\n", f["Date"])
		fmt.Printf("Fingerprint: %v\n", f["Fingerprint"])
		fmt.Printf("Link:        %v\n", f["Link"])
	}

	return result
}
