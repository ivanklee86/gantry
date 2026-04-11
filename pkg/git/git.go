package git

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
)

// FileContent holds a file's path and raw content from a cloned repository.
type FileContent struct {
	Path    string
	Content []byte
}

// CloneOptions configures how a repository is cloned and authenticated.
type CloneOptions struct {
	URL string

	// SSH auth (used for git@ or ssh:// URLs).
	SSHKeyPath     string // path to private key file; if empty, uses SSH agent
	SSHKeyPassword string // passphrase for encrypted SSH key

	// HTTP auth (used for http:// or https:// URLs).
	Token    string // bearer token; takes precedence over Username/Password
	Username string
	Password string
}

// Repository provides read access to a cloned git repository.
type Repository interface {
	GetFiles(paths []string) ([]FileContent, error)
}

type inMemoryRepository struct {
	fs billy.Filesystem
}

// Clone clones a repository into memory and returns a Repository for reading files.
// progress may be nil.
func Clone(opts CloneOptions, progress io.Writer) (Repository, error) {
	auth, err := buildAuth(opts)
	if err != nil {
		return nil, fmt.Errorf("build auth: %w", err)
	}

	fs := memfs.New()
	_, err = gogit.Clone(memory.NewStorage(), fs, &gogit.CloneOptions{
		URL:      opts.URL,
		Auth:     auth,
		Progress: progress,
	})
	if err != nil {
		return nil, fmt.Errorf("clone %s: %w", opts.URL, err)
	}

	return &inMemoryRepository{fs: fs}, nil
}

func buildAuth(opts CloneOptions) (transport.AuthMethod, error) {
	if isSSH(opts.URL) {
		if opts.SSHKeyPath != "" {
			return ssh.NewPublicKeysFromFile("git", opts.SSHKeyPath, opts.SSHKeyPassword)
		}
		return ssh.NewSSHAgentAuth("git")
	}

	// HTTP/HTTPS
	if opts.Token != "" {
		return &githttp.TokenAuth{Token: opts.Token}, nil
	}
	if opts.Username != "" || opts.Password != "" {
		return &githttp.BasicAuth{Username: opts.Username, Password: opts.Password}, nil
	}
	return nil, nil
}

func isSSH(url string) bool {
	return strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://")
}

// GetFiles returns the contents of the given paths from the cloned repository.
// Returns an error on the first path that cannot be opened or read.
func (r *inMemoryRepository) GetFiles(paths []string) ([]FileContent, error) {
	results := make([]FileContent, 0, len(paths))
	for _, path := range paths {
		f, err := r.fs.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		content, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		results = append(results, FileContent{Path: path, Content: content})
	}
	return results, nil
}
