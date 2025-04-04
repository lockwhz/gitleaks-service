package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	git "github.com/go-git/go-git/v5"
	httpAuth "github.com/go-git/go-git/v5/plumbing/transport/http"
	"yourproject/internal/logger"
	"yourproject/internal/vault"
)

// GoGitClient implementa GitClient usando go-git.
type GoGitClient struct {
	Vault vault.VaultClient
}

func (c *GoGitClient) CloneRepo(repoURL string) (string, error) {
	start := time.Now()
	defer logger.Trace("CloneRepo", start)

	creds, err := c.Vault.GetGitHubCredentials()
	if err != nil {
		return "", fmt.Errorf("erro ao recuperar credenciais do GitHub: %v", err)
	}

	dir := filepath.Join(os.TempDir(), fmt.Sprintf("repo_%d", time.Now().UnixNano()))
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: repoURL,
		Auth: &httpAuth.BasicAuth{
			Username: creds.Username,
			Password: creds.Token,
		},
		Progress: os.Stdout,
	})
	if err != nil {
		return "", fmt.Errorf("git clone falhou: %v", err)
	}
	return dir, nil
}
