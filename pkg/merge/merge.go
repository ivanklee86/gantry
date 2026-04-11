package merge

import (
	"fmt"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/ivanklee86/gantry/pkg/git"
)

// Merge evaluates two or more FileContent values and merges them using
// Jsonnet's + operator applied left-to-right, returning canonical JSON.
//
// Files may be plain JSON (.json) or Jsonnet (.jsonnet); both are valid
// Jsonnet. Each file is identified by its Path field, which is used as the
// virtual import key. At least two FileContent values must be provided.
func Merge(files []git.FileContent) ([]byte, error) {
	if len(files) < 2 {
		return nil, fmt.Errorf("merge requires at least 2 files, got %d", len(files))
	}

	seen := make(map[string]struct{}, len(files))
	data := make(map[string]jsonnet.Contents, len(files))
	for i, f := range files {
		if f.Path == "" {
			return nil, fmt.Errorf("file at index %d has empty Path", i)
		}
		if _, ok := seen[f.Path]; ok {
			return nil, fmt.Errorf("duplicate file path %q at index %d", f.Path, i)
		}
		seen[f.Path] = struct{}{}
		data[f.Path] = jsonnet.MakeContents(string(f.Content))
	}

	parts := make([]string, len(files))
	for i, f := range files {
		parts[i] = fmt.Sprintf("(import '%s')", f.Path)
	}
	snippet := strings.Join(parts, " + ")

	vm := jsonnet.MakeVM()
	vm.Importer(&jsonnet.MemoryImporter{Data: data})

	result, err := vm.EvaluateAnonymousSnippet("merge", snippet)
	if err != nil {
		return nil, fmt.Errorf("jsonnet evaluation: %w", err)
	}

	return []byte(result), nil
}
