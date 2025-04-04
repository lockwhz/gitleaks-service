package config

import (
	"os"
	"strconv"
)

type Config struct {
	SQSQueueURL   string // URL da fila SQS.
	PGHost        string // Host do RDS.
	PGPort        string // Porta do RDS.
	PGName        string // Nome do banco.
	PGUser        string // Usuário.
	PGPassword    string // Senha.
	GitleaksPath  string // Caminho do binário do Gitleaks.
	EnableVault   bool   // Habilita integração com Vault para GitHub.
	EnableSecrets bool   // Habilita AWS Secrets Manager para outras credenciais.
	EnableClone   bool   // Habilita clonagem do repositório.
	EnableScan    bool   // Habilita execução do scanner (Gitleaks).
	EnableSQS     bool   // Habilita consumo de mensagens da SQS.
}

func Load() Config {
	parseBool := func(key string) bool {
		val, err := strconv.ParseBool(os.Getenv(key))
		if err != nil {
			return false
		}
		return val
	}
	return Config{
		SQSQueueURL:   os.Getenv("SQS_QUEUE_URL"),
		PGHost:        os.Getenv("PG_HOST"),
		PGPort:        os.Getenv("PG_PORT"),
		PGName:        os.Getenv("PG_NAME"),
		PGUser:        os.Getenv("PG_USER"),
		PGPassword:    os.Getenv("PG_PASSWORD"),
		GitleaksPath:  os.Getenv("GITLEAKS_PATH"),
		EnableVault:   parseBool("ENABLE_VAULT"),
		EnableSecrets: parseBool("ENABLE_SECRETS_MANAGER"),
		EnableClone:   parseBool("ENABLE_GIT_CLONE"),
		EnableScan:    parseBool("ENABLE_GITLEAKS"),
		EnableSQS:     parseBool("ENABLE_SQS"),
	}
}

func (c Config) PostgresConnString() string {
	// Exemplo: "host=localhost port=5432 dbname=mydb user=myuser password=mypass sslmode=disable"
	return "host=" + c.PGHost + " port=" + c.PGPort + " dbname=" + c.PGName + " user=" + c.PGUser + " password=" + c.PGPassword + " sslmode=disable"
}
