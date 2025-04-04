package secrets

import (
	"fmt"
	"os"
)

// SecretsManager define a interface para recuperar segredos.
type SecretsManager interface {
	GetSecret(secretName string) (string, error)
}

// DefaultSecretsManager é uma implementação que lê segredos das variáveis de ambiente.
type DefaultSecretsManager struct{}

func (s *DefaultSecretsManager) GetSecret(secretName string) (string, error) {
	secret := os.Getenv(secretName)
	if secret == "" {
		return "", fmt.Errorf("segredo %s não encontrado", secretName)
	}
	return secret, nil
}
