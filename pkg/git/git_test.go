package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v6/memfs"
	gogit "github.com/go-git/go-git/v6"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check that inMemoryRepository satisfies Repository.
var _ Repository = (*inMemoryRepository)(nil)

// --- buildAuth ---

func TestBuildAuth_HTTP_NoCredentials(t *testing.T) {
	auth, err := buildAuth(CloneOptions{URL: "https://github.com/org/repo.git"})
	require.NoError(t, err)
	assert.Nil(t, auth)
}

func TestBuildAuth_HTTP_Token(t *testing.T) {
	auth, err := buildAuth(CloneOptions{URL: "https://github.com/org/repo.git", Token: "mytoken"})
	require.NoError(t, err)
	require.IsType(t, &githttp.TokenAuth{}, auth)
	assert.Equal(t, "mytoken", auth.(*githttp.TokenAuth).Token)
}

func TestBuildAuth_HTTP_BasicAuth(t *testing.T) {
	auth, err := buildAuth(CloneOptions{URL: "https://github.com/org/repo.git", Username: "user", Password: "pass"})
	require.NoError(t, err)
	require.IsType(t, &githttp.BasicAuth{}, auth)
	ba := auth.(*githttp.BasicAuth)
	assert.Equal(t, "user", ba.Username)
	assert.Equal(t, "pass", ba.Password)
}

func TestBuildAuth_HTTP_Token_TakesPrecedence(t *testing.T) {
	auth, err := buildAuth(CloneOptions{
		URL:      "https://github.com/org/repo.git",
		Token:    "tok",
		Username: "user",
		Password: "pass",
	})
	require.NoError(t, err)
	assert.IsType(t, &githttp.TokenAuth{}, auth)
}

func TestBuildAuth_PlaintextHTTP_RefusesCredentials(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts CloneOptions
	}{
		{"token", CloneOptions{URL: "http://example.com/repo.git", Token: "tok"}},
		{"basic", CloneOptions{URL: "http://example.com/repo.git", Username: "u", Password: "p"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildAuth(tc.opts)
			assert.Error(t, err)
		})
	}
}

func TestBuildAuth_GitProtocol_RefusesCredentials(t *testing.T) {
	_, err := buildAuth(CloneOptions{URL: "git://github.com/org/repo.git", Token: "tok"})
	assert.Error(t, err)
}

func TestBuildAuth_SSH_Agent(t *testing.T) {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("SSH_AUTH_SOCK not set — no SSH agent available")
	}
	auth, err := buildAuth(CloneOptions{URL: "git@github.com:org/repo.git"})
	require.NoError(t, err)
	assert.IsType(t, &ssh.PublicKeysCallback{}, auth)
}

func TestBuildAuth_SSH_Scheme_Agent(t *testing.T) {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("SSH_AUTH_SOCK not set — no SSH agent available")
	}
	auth, err := buildAuth(CloneOptions{URL: "ssh://git@github.com/org/repo.git"})
	require.NoError(t, err)
	assert.IsType(t, &ssh.PublicKeysCallback{}, auth)
}

// --- validateURL ---

func TestValidateURL_Allowed(t *testing.T) {
	for _, u := range []string{
		"https://github.com/org/repo.git",
		"http://internal.example.com/repo.git",
		"ssh://git@github.com/org/repo.git",
		"git@github.com:org/repo.git",
		"git://github.com/org/repo.git",
	} {
		assert.NoError(t, validateURL(u), "expected %q to be allowed", u)
	}
}

func TestValidateURL_Rejected(t *testing.T) {
	for _, u := range []string{
		"file:///etc/passwd",
		"file://localhost/home/user/repo",
		"ftp://example.com/repo",
	} {
		assert.Error(t, validateURL(u), "expected %q to be rejected", u)
	}
}

// --- validateCommitHash ---

func TestValidateCommitHash_Valid(t *testing.T) {
	for _, h := range []string{
		"abc1234",      // 7-char short SHA
		"abc1234def56", // 11-char short SHA
		"abc1234def5678901234567890abcdef12345678", // full 40-char SHA
	} {
		assert.NoError(t, validateCommitHash(h), "expected %q to be valid", h)
	}
}

func TestValidateCommitHash_Invalid(t *testing.T) {
	for _, h := range []string{
		"abc123",                    // too short (6 chars)
		"xyz1234",                   // non-hex chars
		"abc 1234",                  // space
		"<script>alert(1)</script>", // injection attempt
		"",
	} {
		assert.Error(t, validateCommitHash(h), "expected %q to be invalid", h)
	}
}

// --- CloneOptions.String ---

func TestCloneOptions_String_RedactsCredentials(t *testing.T) {
	opts := CloneOptions{
		URL:            "https://github.com/org/repo.git",
		Token:          "secret-token",
		Username:       "secret-user",
		Password:       "secret-pass",
		SSHKeyPassword: "secret-key-pass",
		SSHKeyPath:     "/home/ci/.ssh/id_rsa_deploy",
	}
	s := opts.String()
	assert.NotContains(t, s, "secret-token")
	assert.NotContains(t, s, "secret-user")
	assert.NotContains(t, s, "secret-pass")
	assert.NotContains(t, s, "secret-key-pass")
	// Full SSH key path should not appear; only the base filename is shown.
	assert.NotContains(t, s, "/home/ci/.ssh/")
	assert.Contains(t, s, "id_rsa_deploy")
	assert.Contains(t, s, "***")
}

// --- Clone guards ---

func TestClone_CommitHashAndRefName_Error(t *testing.T) {
	_, err := Clone(CloneOptions{
		URL:           "https://github.com/org/repo.git",
		ReferenceName: "refs/heads/main",
		CommitHash:    "abc1234",
	}, nil)
	assert.Error(t, err)
}

func TestClone_InvalidCommitHash_Error(t *testing.T) {
	_, err := Clone(CloneOptions{
		URL:        "https://github.com/org/repo.git",
		CommitHash: "not-a-sha!",
	}, nil)
	assert.Error(t, err)
}

// --- GetFiles ---

func TestGetFiles_Success(t *testing.T) {
	fs := memfs.New()
	f, err := fs.Create("hello.txt")
	require.NoError(t, err)
	_, err = f.Write([]byte("world"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	repo := &inMemoryRepository{fs: fs}
	files, err := repo.GetFiles([]string{"hello.txt"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "hello.txt", files[0].Path)
	assert.Equal(t, []byte("world"), files[0].Content)
}

func TestGetFiles_EmptyPaths(t *testing.T) {
	repo := &inMemoryRepository{fs: memfs.New()}
	files, err := repo.GetFiles([]string{})
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGetFiles_MissingFile(t *testing.T) {
	repo := &inMemoryRepository{fs: memfs.New()}
	_, err := repo.GetFiles([]string{"missing.txt"})
	assert.Error(t, err)
}

func TestGetFiles_WithSubdirectory(t *testing.T) {
	fs := memfs.New()
	require.NoError(t, fs.MkdirAll("configs/base", 0o755))
	f, err := fs.Create("configs/base/file.json")
	require.NoError(t, err)
	_, err = f.Write([]byte(`{"key":"value"}`))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	repo := &inMemoryRepository{fs: fs, subdirectory: "configs/base"}
	files, err := repo.GetFiles([]string{"file.json"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	// Path reflects the caller-supplied name, not the full prefixed path.
	assert.Equal(t, "file.json", files[0].Path)
	assert.Equal(t, []byte(`{"key":"value"}`), files[0].Content)
}

func TestGetFiles_TraversalRejected(t *testing.T) {
	repo := &inMemoryRepository{fs: memfs.New()}
	for _, p := range []string{
		"../../etc/passwd",
		"../secret",
	} {
		_, err := repo.GetFiles([]string{p})
		assert.Error(t, err, "expected traversal path %q to be rejected", p)
	}
}

func TestGetFiles_TraversalWithSubdirRejected(t *testing.T) {
	repo := &inMemoryRepository{fs: memfs.New(), subdirectory: "configs"}
	_, err := repo.GetFiles([]string{"../../../etc/passwd"})
	assert.Error(t, err)
}

// --- ParseURLSubdir ---

func TestParseURLSubdir_HTTPS(t *testing.T) {
	repoURL, subdir, err := ParseURLSubdir("https://github.com/user/repo//configs")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo", repoURL)
	assert.Equal(t, "configs", subdir)
}

func TestParseURLSubdir_HTTPS_Nested(t *testing.T) {
	repoURL, subdir, err := ParseURLSubdir("https://github.com/user/repo//a/b/c")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo", repoURL)
	assert.Equal(t, "a/b/c", subdir)
}

func TestParseURLSubdir_SSH(t *testing.T) {
	repoURL, subdir, err := ParseURLSubdir("git@github.com:user/repo//subdir")
	require.NoError(t, err)
	assert.Equal(t, "git@github.com:user/repo", repoURL)
	assert.Equal(t, "subdir", subdir)
}

func TestParseURLSubdir_GitProtocol(t *testing.T) {
	repoURL, subdir, err := ParseURLSubdir("git://github.com/user/repo//nested/dir")
	require.NoError(t, err)
	assert.Equal(t, "git://github.com/user/repo", repoURL)
	assert.Equal(t, "nested/dir", subdir)
}

func TestParseURLSubdir_NoSubdir(t *testing.T) {
	repoURL, subdir, err := ParseURLSubdir("https://github.com/user/repo")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo", repoURL)
	assert.Equal(t, "", subdir)
}

func TestParseURLSubdir_SCP_NoSubdir(t *testing.T) {
	repoURL, subdir, err := ParseURLSubdir("git@github.com:user/repo.git")
	require.NoError(t, err)
	assert.Equal(t, "git@github.com:user/repo.git", repoURL)
	assert.Equal(t, "", subdir)
}

func TestParseURLSubdir_TrailingSlashes(t *testing.T) {
	// "//" with nothing after it — subdir is empty, not an error.
	repoURL, subdir, err := ParseURLSubdir("https://github.com/user/repo//")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo", repoURL)
	assert.Equal(t, "", subdir)
}

func TestParseURLSubdir_TraversalRejected(t *testing.T) {
	for _, u := range []string{
		"https://github.com/user/repo//../../etc",
		"https://github.com/user/repo//../secret",
		"git@github.com:user/repo//../../etc",
	} {
		_, _, err := ParseURLSubdir(u)
		assert.Error(t, err, "expected traversal URL %q to be rejected", u)
	}
}

// --- IsLocalPath ---

func TestIsLocalPath(t *testing.T) {
	local := []string{
		"/absolute/path",
		"./relative",
		"../parent",
		"file:///path/to/repo",
	}
	for _, p := range local {
		assert.True(t, IsLocalPath(p), "expected %q to be a local path", p)
	}

	remote := []string{
		"https://github.com/user/repo",
		"git@github.com:user/repo.git",
		"ssh://git@github.com/user/repo",
		"git://github.com/user/repo",
	}
	for _, p := range remote {
		assert.False(t, IsLocalPath(p), "expected %q to be a remote path", p)
	}
}

// --- resolveLocalPath ---

func TestResolveLocalPath_Absolute(t *testing.T) {
	got, err := resolveLocalPath("/workspaces/gantry")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
}

func TestResolveLocalPath_FileURI(t *testing.T) {
	got, err := resolveLocalPath("file:///workspaces/gantry")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
}

func TestResolveLocalPath_Relative(t *testing.T) {
	got, err := resolveLocalPath("../../")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
}

// --- Integration tests ---

func TestIntegration_Clone_HTTPS_Public(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	repo, err := Clone(CloneOptions{
		URL: "https://github.com/ivanklee86/devcontainers",
	}, nil)
	require.NoError(t, err)

	files, err := repo.GetFiles([]string{
		"devcontainer_configs/bases/python/devcontainer.json",
	})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "devcontainer_configs/bases/python/devcontainer.json", files[0].Path)
	assert.NotEmpty(t, files[0].Content)
}

func TestIntegration_Clone_HTTPS_SubDir(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	repo, err := Clone(CloneOptions{
		URL:          "https://github.com/ivanklee86/devcontainers",
		Subdirectory: "devcontainer_configs/bases/python",
	}, nil)
	require.NoError(t, err)

	files, err := repo.GetFiles([]string{"devcontainer.json"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "devcontainer.json", files[0].Path)
	assert.NotEmpty(t, files[0].Content)
}

func TestIntegration_Clone_HTTPS_CommitSHA(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	const repoURL = "https://github.com/ivanklee86/devcontainers"
	const testFile = "devcontainer_configs/bases/python/devcontainer.json"

	// Discover the current HEAD SHA via a shallow clone.
	shallowFS := memfs.New()
	shallowRepo, err := gogit.Clone(memory.NewStorage(), shallowFS, &gogit.CloneOptions{
		URL:   repoURL,
		Depth: 1,
	})
	require.NoError(t, err)
	head, err := shallowRepo.Head()
	require.NoError(t, err)
	headSHA := head.Hash().String()

	// Clone HEAD by full SHA — content must match the shallow clone.
	headRepo, err := Clone(CloneOptions{URL: repoURL}, nil)
	require.NoError(t, err)
	headFiles, err := headRepo.GetFiles([]string{testFile})
	require.NoError(t, err)

	shaRepo, err := Clone(CloneOptions{
		URL:        repoURL,
		CommitHash: headSHA,
	}, nil)
	require.NoError(t, err)

	shaFiles, err := shaRepo.GetFiles([]string{testFile})
	require.NoError(t, err)
	assert.Equal(t, headFiles[0].Content, shaFiles[0].Content)
}

func TestIntegration_OpenLocal_Absolute(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	repo, err := OpenLocal("/workspaces/gantry", "")
	require.NoError(t, err)

	files, err := repo.GetFiles([]string{"go.mod"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "go.mod", files[0].Path)
	assert.NotEmpty(t, files[0].Content)
}

func TestIntegration_OpenLocal_Relative(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	// Tests run from pkg/git, so ../../ is the repo root.
	repo, err := OpenLocal("../../", "")
	require.NoError(t, err)

	files, err := repo.GetFiles([]string{"go.mod"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.NotEmpty(t, files[0].Content)
}

func TestIntegration_OpenLocal_FileURI(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	repo, err := OpenLocal("file:///workspaces/gantry", "")
	require.NoError(t, err)

	files, err := repo.GetFiles([]string{"go.mod"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.NotEmpty(t, files[0].Content)
}
