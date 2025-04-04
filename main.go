package main

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"yourproject/config"
	"yourproject/internal/db"
	"yourproject/internal/git"
	"yourproject/internal/logger"
	"yourproject/internal/services"
	"yourproject/internal/vault"
	"yourproject/internal/scan"
	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go-v2/config"
	awsSQS "github.com/aws/aws-sdk-go-v2/service/sqs"
)

var (
	cloneMaxConc = 3 // Limite de clones simultâneos.
	numWorkers   = 5 // Número de workers no consumer.
)

func main() {
	// Inicializa o logger.
	if err := logger.Init(); err != nil {
		logger.Log.Fatalf("Erro fatal ao iniciar o logger: %v", err)
	}
	defer logger.Log.Sync()

	start := time.Now()
	logger.Trace("main", start)

	// Carrega configurações.
	cfg := config.Load()
	if cfg.GitleaksPath == "" {
		cfg.GitleaksPath = "/usr/local/bin/gitleaks"
	}

	// Conecta ao banco RDS.
	dbConn, err := sql.Open("postgres", cfg.PostgresConnString())
	if err != nil {
		logger.Log.Fatalf("Erro fatal ao conectar ao banco: %v", err)
	}
	defer dbConn.Close()
	store := &db.RDSStore{DB: dbConn}

	// Instancia o VaultClient. Se o Vault estiver desabilitado, use NoOp.
	var vaultClient vault.VaultClient
	if cfg.EnableVault {
		vaultClient = &vault.DefaultVaultClient{}
	} else {
		vaultClient = &vault.NoOpVaultClient{}
	}

	// Instancia o GitClient usando go-git.
	gitClient := &git.GoGitClient{Vault: vaultClient}

	// Instancia o Scanner (Gitleaks) com o caminho configurado.
	scanner := &scan.GitleaksScanner{GitleaksPath: cfg.GitleaksPath}

	// Configura AWS e cria o cliente SQS.
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Log.Fatalf("Erro fatal ao carregar configurações AWS: %v", err)
	}
	sqsClient := awsSQS.NewFromConfig(awsCfg)

	// Instancia o SQSProducer.
	producer := &services.DefaultSQSProducer{
		Client:   sqsClient,
		QueueURL: cfg.SQSQueueURL,
	}

	// Inicia o producer que lê a SQS e produz jobs.
	jobChan := producer.Start()

	// Inicia o consumer que processa os jobs.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer := &services.DefaultJobConsumer{}
		consumer.Start(jobChan, dbConn, gitClient, scanner, cloneMaxConc, numWorkers)
	}()

	wg.Wait()
}
