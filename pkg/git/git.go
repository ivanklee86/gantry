package git

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
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

	// ReferenceName is the branch or tag to clone (e.g. "refs/heads/main").
	// If empty, the remote's default branch is used.
	ReferenceName string

	// SSH auth (used for git@ or ssh:// URLs).
	SSHKeyPath     string // path to private key file; if empty, uses SSH agent
	SSHKeyPassword string // passphrase for encrypted SSH key

	// HTTP auth (used for http:// or https:// URLs).
	Token    string // bearer token; takes precedence over Username/Password
	Username string
	Password string
}

// String returns a representation of CloneOptions with sensitive fields redacted.
func (o CloneOptions) String() string {
	// Use an unexported mirror type to avoid infinite recursion via the Stringer interface.
	type safe struct {
		URL, ReferenceName, SSHKeyPath, SSHKeyPassword, Token, Username, Password string
	}
	mask := func(s string) string {
		if s != "" {
			return "***"
		}
		return ""
	}
	return fmt.Sprintf("%+v", safe{
		URL:            o.URL,
		ReferenceName:  o.ReferenceName,
		SSHKeyPath:     o.SSHKeyPath,
		SSHKeyPassword: mask(o.SSHKeyPassword),
		Token:          mask(o.Token),
		Username:       o.Username,
		Password:       mask(o.Password),
	})
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
	if err := validateURL(opts.URL); err != nil {
		return nil, err
	}

	auth, err := buildAuth(opts)
	if err != nil {
		return nil, fmt.Errorf("build auth: %w", err)
	}

	cloneOpts := &gogit.CloneOptions{
		URL:      opts.URL,
		Auth:     auth,
		Depth:    1,
		Progress: progress,
	}
	if opts.ReferenceName != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(opts.ReferenceName)
		cloneOpts.SingleBranch = true
	}

	fs := memfs.New()
	_, err = gogit.Clone(memory.NewStorage(), fs, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("clone %s: %w", redactURL(opts.URL), err)
	}

	return &inMemoryRepository{fs: fs}, nil
}

func validateURL(rawURL string) error {
	// SCP-like SSH URLs (git@host:path) have no parseable scheme.
	if strings.HasPrefix(rawURL, "git@") {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	switch u.Scheme {
	case "https", "http", "ssh":
		return nil
	default:
		return fmt.Errorf("unsupported URL scheme %q: allowed schemes are https, http, ssh, and git@", u.Scheme)
	}
}

func redactURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "git@") {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid-url>"
	}
	return u.Redacted()
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
		content, err := func() ([]byte, error) {
			f, err := r.fs.Open(path)
			if err != nil {
				return nil, fmt.Errorf("open %s: %w", path, err)
			}
			data, readErr := io.ReadAll(f)
			if closeErr := f.Close(); closeErr != nil && readErr == nil {
				return nil, fmt.Errorf("close %s: %w", path, closeErr)
			}
			return data, readErr
		}()
		if err != nil {
			return nil, err
		}
		results = append(results, FileContent{Path: path, Content: content})
	}
	return results, nil
}
