package gantry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acarl005/stripansi"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/ivanklee86/gantry/pkg/config"
	"github.com/ivanklee86/gantry/pkg/git"
	"github.com/ivanklee86/gantry/pkg/merger"
)

// openRepoFunc is the function used by Build to open a repository.
// It is a package-level variable so tests can substitute a mock.
var openRepoFunc = (*Gantry).openRepo

// Build executes the build pipeline described by cfg:
//  1. For each overlay, clone or open the repository and fetch the specified files.
//  2. Merge all fetched files in order using Jsonnet.
//  3. Write the result to cfg.OutputPath (if cfg.Write) or print it to g.Out.
func (g *Gantry) Build(_ context.Context, cfg config.BuildConfig) error {
	target := "stdout"
	if cfg.Write {
		target = stripansi.Strip(cfg.OutputPath)
	}
	printToStreamWithColor(g.Err, text.FgHiCyan, fmt.Sprintf("🚀 Starting to build devcontainer with %d overlay(s) → %s", len(cfg.Overlays), target))

	var allFiles []git.FileContent

	for i, overlay := range cfg.Overlays {
		printToStreamWithColor(g.Err, text.FgHiCyan, fmt.Sprintf("🔗 Overlay %d/%d: %s", i+1, len(cfg.Overlays), stripansi.Strip(overlay.Repo)))

		repo, err := openRepoFunc(g, overlay)
		if err != nil {
			return fmt.Errorf("overlay %d: open repo: %w", i+1, err)
		}

		files, err := repo.GetFiles(overlay.Files)
		if err != nil {
			return fmt.Errorf("overlay %d: get files: %w", i+1, err)
		}

		allFiles = append(allFiles, files...)
	}

	merged, err := merger.Merge(allFiles)
	if err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	if cfg.Write {
		if err := writeOutput(cfg.OutputPath, merged); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		printToStreamWithColor(g.Err, text.FgHiGreen, fmt.Sprintf("✅ Wrote %s", stripansi.Strip(cfg.OutputPath)))
	} else {
		g.Output(merged)
	}

	return nil
}

// openRepo resolves an overlay's repository reference to a git.Repository.
// Local paths are opened with git.OpenLocal; all other values are cloned.
func (g *Gantry) openRepo(o config.Overlay) (git.Repository, error) {
	if git.IsLocalPath(o.Repo) {
		return git.OpenLocal(o.Repo, o.Subdirectory)
	}

	repoURL, subdir, err := git.ParseURLSubdir(o.Repo)
	if err != nil {
		return nil, err
	}
	// Subdirectory field takes precedence over the // URL syntax.
	if o.Subdirectory != "" {
		subdir = o.Subdirectory
	}

	opts := git.CloneOptions{
		URL:            repoURL,
		ReferenceName:  o.Ref,
		CommitHash:     o.Commit,
		Subdirectory:   subdir,
		Token:          o.Auth.Token,
		Username:       o.Auth.Username,
		Password:       o.Auth.Password,
		SSHKeyPath:     o.Auth.SSHKeyPath,
		SSHKeyPassword: o.Auth.SSHKeyPassword,
	}

	return git.Clone(opts, g.Err)
}

// writeOutput creates parent directories as needed and writes content to path.
func writeOutput(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
