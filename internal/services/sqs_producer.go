package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    gitclient "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/git"
    "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/logger"
    "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/queue"
    "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/services"

    "github.com/google/uuid"
)

func main() {
    /* ────────── 1. logger & ctx com cancel em SIGTERM ────────── */
    logger.InitLogger()
    slog := logger.GetSugaredLogger()

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    /* ────────── 2. SQS consumer ────────── */
    queueURL := os.Getenv("SQS_URL")
    region := os.Getenv("AWS_REGION")
    consumer, err := queue.NewConsumer(ctx, queueURL, region)
    if err != nil {
        slog.Fatalf("sqs consumer: %v", err)
    }

    /* ────────── 3. DB connection (reutilizada por todos os jobs) ────────── */
    dbSvc := services.NewDatabase()
    if _, err := dbSvc.Connect(
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASS"),
    ); err != nil {
        slog.Fatalf("db connect: %v", err)
    }
    defer dbSvc.Close()

    accessToken := os.Getenv("GITHUB_TOKEN") // opcional

    /* ────────── 4. Loop de processamento ────────── */
    if err := consumer.RunLoop(ctx, func(ctx context.Context, job queue.Job) error {
        scanID, err := uuid.Parse(job.ScanID)
        if err != nil {
            slog.Errorf("scanID inválido: %v", err)
            return err
        }
        repoID, err := uuid.Parse(job.RepoID)
        if err != nil {
            slog.Errorf("repoID inválido: %v", err)
            return err
        }

        slog.Infof("processando repo %s (scan %s)", job.RepoURL, scanID)

        // 1) clone + scan
        result, err := gitclient.ScanRepo(job.RepoURL, accessToken)
        if err != nil {
            slog.Errorf("scan repo: %v", err)
            return err
        }

        // 2) persistir findings
        if err := dbSvc.SaveFindings(ctx, scanID, repoID, result.Results); err != nil {
            slog.Errorf("save findings: %v", err)
            return err
        }

        slog.Infof("OK – %d branches, %d leaks únicos", len(result.Results), len(resultscan.FlattenFindings(result.Results)))
        return nil
    }); err != nil {
        slog.Fatalf("run loop: %v", err)
    }

    slog.Info("encerrado com sucesso")
}
