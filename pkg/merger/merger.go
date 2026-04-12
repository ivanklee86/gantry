package merger

import (
	"fmt"
	"strings"

	"github.com/google/go-jsonnet"

	"github.com/ivanklee86/gantry/pkg/git"
)

// Merge merges the given file contents in order using the Jsonnet + operator,
// which performs a shallow object merge where later entries override earlier ones.
// Each entry may be plain JSON (valid Jsonnet) or a Jsonnet expression.
// The result is returned as a pretty-printed JSON string.
//
// Filesystem imports inside Jsonnet files are disabled; all content must be
// provided via the files slice.
func Merge(files []git.FileContent) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files to merge")
	}

	vm := newVM(files)

	// Build a Jsonnet snippet that imports each file by its virtual path and
	// merges them left-to-right with the + operator.
	//
	// Example for three files a.json, b.jsonnet, c.json:
	//   (import "a.json") + (import "b.jsonnet") + (import "c.json")
	parts := make([]string, len(files))
	for i, f := range files {
		parts[i] = fmt.Sprintf(`(import %q)`, f.Path)
	}
	snippet := strings.Join(parts, " + ")

	result, err := vm.EvaluateAnonymousSnippet("<merge>", snippet)
	if err != nil {
		return "", fmt.Errorf("jsonnet evaluation: %w", err)
	}
	return result, nil
}

// newVM creates a Jsonnet VM pre-loaded with the given file contents as virtual
// imports. The VM rejects any import path not present in the provided files.
func newVM(files []git.FileContent) *jsonnet.VM {
	vm := jsonnet.MakeVM()

	// Build an in-memory importer from the provided file contents.
	contents := make(map[string]jsonnet.Contents, len(files))
	for _, f := range files {
		contents[f.Path] = jsonnet.MakeContents(string(f.Content))
	}
	vm.Importer(&memImporter{contents: contents})

	return vm
}

// memImporter is a jsonnet.Importer that serves only from an in-memory map.
// Any import path not present in the map returns an error, preventing
// arbitrary filesystem reads from within Jsonnet files.
type memImporter struct {
	contents map[string]jsonnet.Contents
}

func (m *memImporter) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	c, ok := m.contents[importedPath]
	if !ok {
		return jsonnet.Contents{}, "", fmt.Errorf(
			"import %q is not available: only pre-fetched files may be imported", importedPath,
		)
	}
	return c, importedPath, nil
}
