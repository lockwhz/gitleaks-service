package git

// GitClient define a interface para operações de clonagem.
type GitClient interface {
	CloneRepo(repoURL string) (string, error)
}
