package secrets

import (
	"context"
	"fmt"
	"os"

	"yourproject/internal/logger"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type AWSSecretFetcher struct {
	client *secretsmanager.Client
	region string
}

func NewAWSSecretFetcher(region string) (*AWSSecretFetcher, error) {
	log := logger.GetSugaredLogger()

	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			log.Error("Região AWS não definida nem como argumento nem via variável de ambiente AWS_REGION")
			return nil, fmt.Errorf("AWS region não definida")
		}
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		log.Errorf("Erro ao carregar configuração AWS: %v", err)
		return nil, err
	}

	client := secretsmanager.NewFromConfig(cfg)

	return &AWSSecretFetcher{
		client: client,
		region: region,
	}, nil
}

func (f *AWSSecretFetcher) GetSecret(secretID string) (string, error) {
	log := logger.GetSugaredLogger()

	if secretID == "" {
		secretID = os.Getenv("DB_SM_SECRET_ID")
		if secretID == "" {
			log.Error("ID do secret não definido nem como argumento nem via env DB_SM_SECRET_ID")
			return "", fmt.Errorf("ID do secret não fornecido")
		}
	}

	resp, err := f.client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		log.Errorf("Erro ao buscar secret '%s': %v", secretID, err)
		return "", err
	}

	if resp.SecretString == nil {
		log.Warnf("Secret '%s' não possui conteúdo em SecretString", secretID)
		return "", fmt.Errorf("secret vazio para '%s'", secretID)
	}

	log.Infof("Secret '%s' obtido com sucesso", secretID)
	return *resp.SecretString, nil
}
