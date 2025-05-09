// Package git executa o clone do repositório, roda o Gitleaks em cada
// branch remoto e devolve resultados fortemente tipados.
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
    "time"
)

/* ============================== Tipos ============================== */

type Finding struct {
    BranchName  string `json:"branch_name"`
    File        string `json:"File"`
    StartLine   int    `json:"StartLine"`
    RuleID      string `json:"RuleID"`
    Fingerprint string `json:"Fingerprint"`
    Date        string `json:"Date"` // ISO‑8601
}

type BranchScanResult struct {
    BranchName string    `json:"branch_name"`
    Success    bool      `json:"success"`
    Findings   []Finding `json:"findings,omitempty"`
    Error      string    `json:"error,omitempty"`
}

type RepoScanResult struct {
    Repository string             `json:"repository"`
    Results    []BranchScanResult `json:"results"`
}

/* ============================ API pública =========================== */

// ScanRepo clona cada branch remoto e executa o Gitleaks individualmente.
// repoURL deve terminar com .git; accessToken pode ser vazio para clone público.
func ScanRepo(repoURL, accessToken string) (*RepoScanResult, error) {
    branches, err := listRemoteBranches(repoURL)
    if err != nil {
        return nil, fmt.Errorf("listar branches: %w", err)
    }

    baseDir := filepath.Join(os.TempDir(), slugify(filepath.Base(repoURL))+"-scan-"+time.Now().Format("20060102150405"))
    if err := os.MkdirAll(baseDir, 0o755); err != nil {
        return nil, fmt.Errorf("mkdir tmp: %w", err)
    }
    defer os.RemoveAll(baseDir)

    var results []BranchScanResult
    for _, branch := range branches {
        branchDir := filepath.Join(baseDir, strings.ReplaceAll(branch, "/", "-"))
        cloneURL := repoURL
        if accessToken != "" {
            cloneURL = fmt.Sprintf("https://x-access-token:%s@%s", accessToken, strings.TrimPrefix(repoURL, "https://"))
        }

        if err := exec.Command("git", "clone", "--depth", "1", "--branch", branch, cloneURL, branchDir).Run(); err != nil {
            results = append(results, BranchScanResult{BranchName: branch, Success: false, Error: err.Error()})
            continue
        }

        results = append(results, runGitleaks(branchDir, branch))
    }

    return &RepoScanResult{Repository: repoURL, Results: results}, nil
}

/* ========================== Implementação ========================== */

func listRemoteBranches(repoURL string) ([]string, error) {
    cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
    var out bytes.Buffer
    cmd.Stdout = &out
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("ls-remote: %w", err)
    }

    var branches []string
    scanner := bufio.NewScanner(&out)
    for scanner.Scan() {
        parts := strings.Split(scanner.Text(), "\t")
        if len(parts) == 2 {
            ref := strings.TrimPrefix(parts[1], "refs/heads/")
            if ref != "" {
                branches = append(branches, ref)
            }
        }
    }
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("scanner: %w", err)
    }
    return branches, nil
}

func runGitleaks(repoPath, branchName string) BranchScanResult {
    const report = "gitleaks_report.json"

    cmd := exec.Command("gitleaks", "detect", "-s", repoPath, "-r", report, "--report-format", "json")
    if err := cmd.Run(); err != nil {
        if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
            return BranchScanResult{BranchName: branchName, Success: false, Error: err.Error()}
        }
    }

    data, err := os.ReadFile(report)
    _ = os.Remove(report)
    if err != nil {
        return BranchScanResult{BranchName: branchName, Success: false, Error: fmt.Sprintf("read report: %v", err)}
    }

    var findings []Finding
    if len(data) > 0 {
        if err := json.Unmarshal(data, &findings); err != nil {
            return BranchScanResult{BranchName: branchName, Success: false, Error: fmt.Sprintf("decode json: %v", err)}
        }
        // 🔹 Injeta o nome da branch em cada finding
        for i := range findings {
            findings[i].BranchName = branchName
            findings[i].Date = ensureISO8601(findings[i].Date)
        }
    }

    return BranchScanResult{BranchName: branchName, Success: true, Findings: findings}
}

/* ============================== Helpers ============================= */

func slugify(s string) string {
    s = strings.ToLower(strings.TrimSuffix(s, ".git"))
    s = strings.Map(func(r rune) rune {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
            return r
        }
        return '-'
    }, s)
    return strings.Trim(s, "-")
}

func ensureISO8601(d string) string {
    if d == "" {
        return time.Now().UTC().Format(time.RFC3339)
    }
    if _, err := time.Parse(time.RFC3339, d); err == nil {
        return d
    }
    if t, err := time.Parse("2006-01-02 15:04:05 -0700", d); err == nil {
        return t.UTC().Format(time.RFC3339)
    }
    return time.Now().UTC().Format(time.RFC3339)
}
