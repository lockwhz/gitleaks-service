package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    gitclient "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/git"
    "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/logger"
    "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/services"

    "github.com/google/uuid"
)

// exemplo fictício de Job utilizado na mensagem SQS
// substitua pelos campos reais
// type Job struct { ScanID string; RepositoryFullName string }

func main() {
    /* ─────────────────────────── 1. logging ─────────────────────────── */
    logger.InitLogger()
    slog := logger.GetSugaredLogger()

    /* ─────────────────────────── 2. obter mensagem SQS ──────────────── */
    // Aqui deve entrar o código que consome a fila e preenche a struct Job.
    // Para fins de exemplo, uso valores mockados:
    job := struct {
        ScanID             string
        RepositoryFullName string
    }{
        ScanID:             uuid.NewString(),
        RepositoryFullName: "org/repo",
    }

    repoURL := fmt.Sprintf("https://github.com/%s.git", job.RepositoryFullName)

    /* ─────────────────────────── 3. executa scan Git/Gitleaks ───────── */
    accessToken := os.Getenv("GITHUB_TOKEN") // opcional
    result, err := gitclient.ScanRepo(repoURL, accessToken)
    if err != nil {
        slog.Fatalf("scan repo: %v", err)
    }

    /* ─────────────────────────── 4. persiste resultados ─────────────── */
    dbSvc := services.NewDatabase()
    if _, err := dbSvc.Connect(
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASS"),
    ); err != nil {
        slog.Fatalf("connect db: %v", err)
    }
    defer dbSvc.Close()

    scanID := uuid.MustParse(job.ScanID)
    repoID := uuid.New() // ou obtenha de tabela de repositórios

    if err := dbSvc.SaveFindings(context.Background(), scanID, repoID, result.Results); err != nil {
        slog.Fatalf("save findings: %v", err)
    }

    /* ─────────────────────────── 5. log/retorno opcional ────────────── */
    if pretty, err := json.MarshalIndent(result, "", "  "); err == nil {
        log.Println(string(pretty))
    }
}
