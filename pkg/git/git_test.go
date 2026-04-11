package git

import (
	"os"
	"testing"

	"github.com/go-git/go-billy/v6/memfs"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check that inMemoryRepository satisfies Repository.
var _ Repository = (*inMemoryRepository)(nil)

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

func TestValidateURL_Allowed(t *testing.T) {
	for _, u := range []string{
		"https://github.com/org/repo.git",
		"http://internal.example.com/repo.git",
		"ssh://git@github.com/org/repo.git",
		"git@github.com:org/repo.git",
	} {
		assert.NoError(t, validateURL(u), "expected %q to be allowed", u)
	}
}

func TestValidateURL_Rejected(t *testing.T) {
	for _, u := range []string{
		"file:///etc/passwd",
		"file://localhost/home/user/repo",
		"git://github.com/org/repo.git",
		"ftp://example.com/repo",
	} {
		assert.Error(t, validateURL(u), "expected %q to be rejected", u)
	}
}

func TestCloneOptions_String_RedactsCredentials(t *testing.T) {
	opts := CloneOptions{
		URL:            "https://github.com/org/repo.git",
		Token:          "secret-token",
		Password:       "secret-pass",
		SSHKeyPassword: "secret-key-pass",
	}
	s := opts.String()
	assert.NotContains(t, s, "secret-token")
	assert.NotContains(t, s, "secret-pass")
	assert.NotContains(t, s, "secret-key-pass")
	assert.Contains(t, s, "***")
}

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
