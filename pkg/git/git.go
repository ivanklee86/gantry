package git

import (
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
)

// validSHA matches a full (40-char) or short (7–40 char) hex commit SHA.
var validSHA = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

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
	// Cannot be combined with CommitHash.
	ReferenceName string

	// CommitHash, if set, checks out this exact commit after cloning (full 40-char SHA only).
	// Requires a full clone — Depth is set to 0 (unlimited) automatically.
	// Cannot be combined with ReferenceName.
	CommitHash string

	// Subdirectory, if set, is prepended to all paths in GetFiles.
	// Must not contain ".." components; use ParseURLSubdir to extract from a URL.
	Subdirectory string

	// SSH auth (used for git@ or ssh:// URLs).
	SSHKeyPath     string // path to private key file; if empty, uses SSH agent
	SSHKeyPassword string // passphrase for encrypted SSH key

	// HTTP auth (used for https:// URLs only — credentials are never sent over http://).
	Token    string // bearer token; takes precedence over Username/Password
	Username string
	Password string
}

// String returns a representation of CloneOptions with sensitive fields redacted.
func (o CloneOptions) String() string {
	// Use an unexported mirror type to avoid infinite recursion via the Stringer interface.
	type safe struct {
		URL            string
		ReferenceName  string
		CommitHash     string
		Subdirectory   string
		SSHKeyPath     string
		SSHKeyPassword string
		Token          string
		Username       string
		Password       string
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
		CommitHash:     o.CommitHash,
		Subdirectory:   o.Subdirectory,
		SSHKeyPath:     filepath.Base(o.SSHKeyPath), // omit directory to avoid leaking FS layout
		SSHKeyPassword: mask(o.SSHKeyPassword),
		Token:          mask(o.Token),
		Username:       mask(o.Username),
		Password:       mask(o.Password),
	})
}

// Repository provides read access to a cloned git repository.
type Repository interface {
	GetFiles(paths []string) ([]FileContent, error)
}

type inMemoryRepository struct {
	fs           billy.Filesystem
	subdirectory string
}

// Clone clones a repository into memory and returns a Repository for reading files.
// progress may be nil.
func Clone(opts CloneOptions, progress io.Writer) (Repository, error) {
	if opts.CommitHash != "" && opts.ReferenceName != "" {
		return nil, fmt.Errorf("CommitHash and ReferenceName are mutually exclusive")
	}
	if opts.CommitHash != "" {
		if err := validateCommitHash(opts.CommitHash); err != nil {
			return nil, err
		}
	}
	if err := validateURL(opts.URL); err != nil {
		return nil, err
	}

	auth, err := buildAuth(opts)
	if err != nil {
		return nil, fmt.Errorf("build auth: %w", err)
	}

	// Depth 0 means unlimited (full clone). A full clone is required when checking
	// out an arbitrary commit SHA, since shallow clones only fetch the tip commit.
	depth := 1
	if opts.CommitHash != "" {
		depth = 0
	}

	cloneOpts := &gogit.CloneOptions{
		URL:      opts.URL,
		Auth:     auth,
		Depth:    depth,
		Progress: progress,
	}
	if opts.ReferenceName != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(opts.ReferenceName)
		cloneOpts.SingleBranch = true
	}

	fs := memfs.New()
	gitRepo, err := gogit.Clone(memory.NewStorage(), fs, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("clone %s: %w", redactURL(opts.URL), err)
	}

	if opts.CommitHash != "" {
		wt, err := gitRepo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("get worktree: %w", err)
		}
		if err := wt.Checkout(&gogit.CheckoutOptions{
			Hash: plumbing.NewHash(opts.CommitHash),
		}); err != nil {
			return nil, fmt.Errorf("checkout %.12s: %w", opts.CommitHash, err)
		}
	}

	return &inMemoryRepository{fs: fs, subdirectory: opts.Subdirectory}, nil
}

// IsLocalPath reports whether ref looks like a local filesystem path or file URI
// rather than a remote URL. Use this to decide between Clone and OpenLocal.
// Anything that is not a recognized remote URL scheme (https, http, ssh, git, git@)
// is treated as local, so paths like ".", "..", "../../configs", or absolute paths
// all work without explicit enumeration.
func IsLocalPath(ref string) bool {
	if strings.HasPrefix(ref, "git@") {
		return false
	}
	u, err := url.Parse(ref)
	if err != nil {
		return true
	}
	switch u.Scheme {
	case "https", "http", "ssh", "git":
		return false
	default:
		return true
	}
}

// OpenLocal opens a local git repository for reading.
// path may be an absolute path, a relative path (resolved from CWD), or a file:// URI.
// subdir, if non-empty, is prepended to all paths in GetFiles.
func OpenLocal(repoPath, subdir string) (Repository, error) {
	localPath, err := resolveLocalPath(repoPath)
	if err != nil {
		return nil, err
	}

	gitRepo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("open local repo %s: %w", localPath, err)
	}

	wt, err := gitRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("get worktree: %w", err)
	}

	return &inMemoryRepository{fs: wt.Filesystem, subdirectory: subdir}, nil
}

func resolveLocalPath(p string) (string, error) {
	var abs string
	if strings.HasPrefix(p, "file://") {
		u, err := url.Parse(p)
		if err != nil {
			return "", fmt.Errorf("invalid file URI: %w", err)
		}
		abs = u.Path
	} else if !filepath.IsAbs(p) {
		var err error
		abs, err = filepath.Abs(p)
		if err != nil {
			return "", fmt.Errorf("resolve relative path: %w", err)
		}
	} else {
		abs = p
	}

	// Resolve symlinks to prevent TOCTOU attacks on shared execution environments.
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks for %s: %w", abs, err)
	}
	return real, nil
}

// ParseURLSubdir splits a repository URL on the "//" subdirectory separator.
// e.g. "https://github.com/user/repo//configs" → ("https://github.com/user/repo", "configs", nil)
// Works correctly with https://, ssh://, git://, and git@ URLs.
// Returns an error if the subdirectory contains ".." components or is absolute.
func ParseURLSubdir(rawURL string) (repoURL, subdir string, err error) {
	// Skip past the scheme's "://" so we don't mistake it for the separator.
	searchFrom := 0
	if i := strings.Index(rawURL, "://"); i != -1 {
		searchFrom = i + 3
	}
	if i := strings.Index(rawURL[searchFrom:], "//"); i != -1 {
		sep := searchFrom + i
		subdir = rawURL[sep+2:]
		repoURL = strings.TrimSuffix(rawURL[:sep], "/")
	} else {
		return rawURL, "", nil
	}

	if subdir != "" {
		cleaned := path.Clean(subdir)
		if strings.HasPrefix(cleaned, "..") || path.IsAbs(cleaned) {
			return "", "", fmt.Errorf("subdirectory %q escapes repository root", subdir)
		}
		subdir = cleaned
	}
	return repoURL, subdir, nil
}

func validateCommitHash(h string) error {
	if !validSHA.MatchString(h) {
		return fmt.Errorf("invalid commit SHA: must be 7–40 hex characters")
	}
	return nil
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
	case "https", "http", "ssh", "git":
		return nil
	default:
		return fmt.Errorf("unsupported URL scheme %q: allowed schemes are https, http, ssh, git, and git@", u.Scheme)
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
	hasCredentials := opts.Token != "" || opts.Username != "" || opts.Password != ""

	// Refuse to transmit credentials over plaintext transports.
	if hasCredentials && strings.HasPrefix(opts.URL, "http://") {
		return nil, fmt.Errorf("refusing to send credentials over plaintext http://; use https://")
	}
	// git:// has no authentication mechanism at all — credentials would be silently ignored.
	if hasCredentials && strings.HasPrefix(opts.URL, "git://") {
		return nil, fmt.Errorf("git:// protocol does not support authentication; use https:// or ssh://")
	}

	if isSSH(opts.URL) {
		if opts.SSHKeyPath != "" {
			return ssh.NewPublicKeysFromFile("git", opts.SSHKeyPath, opts.SSHKeyPassword)
		}
		return ssh.NewSSHAgentAuth("git")
	}

	// HTTPS
	if opts.Token != "" {
		return &githttp.TokenAuth{Token: opts.Token}, nil
	}
	if opts.Username != "" || opts.Password != "" {
		return &githttp.BasicAuth{Username: opts.Username, Password: opts.Password}, nil
	}
	return nil, nil
}

func isSSH(u string) bool {
	return strings.HasPrefix(u, "git@") || strings.HasPrefix(u, "ssh://")
}

// maxFileBytes caps the size of a single file read from the repository.
const maxFileBytes = 10 * 1024 * 1024 // 10 MB

// GetFiles returns the contents of the given paths from the repository.
// If a subdirectory was configured, it is prepended to each path when reading;
// FileContent.Path always reflects the caller-supplied path.
// Paths containing ".." components that would escape the repository root are rejected.
// Returns an error on the first path that cannot be opened or read.
func (r *inMemoryRepository) GetFiles(paths []string) ([]FileContent, error) {
	results := make([]FileContent, 0, len(paths))
	for _, p := range paths {
		// securejoin provides containment checking: the result is always within root.
		// We use "/" as the root because billy filesystems are rooted at "/".
		root := "/"
		base := p
		if r.subdirectory != "" {
			base = r.subdirectory + "/" + p
		}
		fullPath, err := securejoin.SecureJoin(root, base)
		if err != nil {
			return nil, fmt.Errorf("path containment check for %q: %w", p, err)
		}
		// Strip the leading "/" added by securejoin for billy compatibility.
		fullPath = strings.TrimPrefix(fullPath, "/")

		content, err := func() ([]byte, error) {
			f, err := r.fs.Open(fullPath)
			if err != nil {
				return nil, fmt.Errorf("open %s: %w", fullPath, err)
			}
			limited := io.LimitReader(f, maxFileBytes+1)
			data, readErr := io.ReadAll(limited)
			if closeErr := f.Close(); closeErr != nil && readErr == nil {
				return nil, fmt.Errorf("close %s: %w", fullPath, closeErr)
			}
			if readErr != nil {
				return nil, readErr
			}
			if int64(len(data)) > maxFileBytes {
				return nil, fmt.Errorf("file %s exceeds maximum allowed size (%d bytes)", fullPath, maxFileBytes)
			}
			return data, nil
		}()
		if err != nil {
			return nil, err
		}
		results = append(results, FileContent{Path: p, Content: content})
	}
	return results, nil
}
