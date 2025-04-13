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
	"yourproject/internal/secrets"
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
	github_access_token_test = "sk-test_9f4d3b8a7c1e4f2092a9c8e12f6b7d5a"
	pk = "pk_live_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiYXBpLXVzZXIxMjMiLCJzY29wZSI6InJlYWQifQ.4Dfkm0v7tUxuHZ9cpjblYv9O95OAD9LPCYiAjyS-Y1M"
	AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
	AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

func main() {
	// Inicializa o logger (singleton).
	if err := logger.Init(); err != nil {
		logger.Log.Fatalf("Erro fatal ao iniciar o logger: %v", err)
	}
	defer logger.Log.Sync()

	start := time.Now()
	logger.Trace("main", start)

	// Carrega as configurações.
	cfg := config.Load()
	if cfg.GitleaksPath == "" {
		cfg.GitleaksPath = "/usr/local/bin/gitleaks"
	}

	// Recupera a senha do banco do AWS Secrets Manager.
	var dbPassword string
	if cfg.EnableSecrets {
		secretFetcher, err := secrets.NewAWSSecretFetcher(cfg.AWSRegion)
		if err != nil {
			logger.Log.Fatalf("Erro fatal ao criar AWSSecretFetcher: %v", err)
		}
		secret, err := secretFetcher.GetSecret(cfg.DBSecretID)
		if err != nil {
			logger.Log.Fatalf("Erro fatal ao recuperar o secret do DB: %v", err)
		}
		dbPassword = secret
	} else {
		dbPassword = os.Getenv("PG_PASSWORD")
	}

	// Atualiza a configuração com a senha recuperada.
	cfg.PGPassword = dbPassword

	// Conecta ao banco RDS.
	dbConn, err := sql.Open("postgres", cfg.PostgresConnString())
	if err != nil {
		logger.Log.Fatalf("Erro fatal ao conectar ao banco: %v", err)
	}
	defer dbConn.Close()
	store := &db.RDSStore{DB: dbConn}

	// Instancia o VaultClient.
	var vaultClient vault.VaultClient
	if cfg.EnableVault {
		vaultClient = &vault.DefaultVaultClient{}
	} else {
		vaultClient = &vault.NoOpVaultClient{}
	}

	// Instancia o GitClient.
	gitClient := &git.GoGitClient{Vault: vaultClient}

	// Instancia o Scanner.
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

	// Inicia o producer que lê da SQS e produz jobs.
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
