package vault

import (
	"fmt"
	"os"
)

// VaultClient define a interface para recuperar credenciais do GitHub.
type VaultClient interface {
	GetGitHubCredentials() (*GitHubCredentials, error)
}

type GitHubCredentials struct {
	Username string
	Token    string
}

// DefaultVaultClient recupera as credenciais do Vault (simulado via variáveis de ambiente).
type DefaultVaultClient struct{}

func (v *DefaultVaultClient) GetGitHubCredentials() (*GitHubCredentials, error) {
	username := os.Getenv("GITHUB_USERNAME")
	token := os.Getenv("GITHUB_TOKEN")
	if username == "" || token == "" {
		return nil, fmt.Errorf("credenciais do GitHub não encontradas")
	}
	return &GitHubCredentials{
		Username: username,
		Token:    token,
	}, nil
}

// NoOpVaultClient retorna credenciais padrão para testes.
type NoOpVaultClient struct{}

func (v *NoOpVaultClient) GetGitHubCredentials() (*GitHubCredentials, error) {
	return &GitHubCredentials{
		Username: "default_user",
		Token:    "default_token",
	}, nil
}
