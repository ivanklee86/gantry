package merge

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/ivanklee86/gantry/pkg/git"
)

// safePath matches only characters that are safe to embed in a quoted Jsonnet
// import expression. Paths with other characters are rejected to prevent
// injection of arbitrary Jsonnet into the evaluated snippet.
var safePath = regexp.MustCompile(`^[a-zA-Z0-9_\-./]+$`)

const maxOutputBytes = 10 * 1024 * 1024 // 10 MB

// Merge evaluates one or more FileContent values and merges them using
// Jsonnet's + operator applied left-to-right, returning canonical JSON.
//
// Files may be plain JSON (.json) or Jsonnet (.jsonnet); both are valid
// Jsonnet. Each file is identified by its Path field, which is used as the
// virtual import key. At least one FileContent value must be provided.
//
// Merge performs a shallow top-level merge: right-hand fields overwrite
// left-hand fields at the top level. Deep merging of nested objects requires
// the overlay Jsonnet file to use the +: operator on those fields.
//
// The MemoryImporter provides isolation: Jsonnet files cannot import from the
// real filesystem or network — only paths provided in files are importable.
// Error values may contain caller-supplied path strings.
func Merge(files []git.FileContent) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("merge requires at least 1 file, got 0")
	}

	data := make(map[string]jsonnet.Contents, len(files))
	for i, f := range files {
		if f.Path == "" {
			return nil, fmt.Errorf("file at index %d has empty Path", i)
		}
		if !safePath.MatchString(f.Path) {
			return nil, fmt.Errorf("file at index %d has invalid path characters: %q", i, f.Path)
		}
		if _, ok := data[f.Path]; ok {
			return nil, fmt.Errorf("duplicate file path %q at index %d", f.Path, i)
		}
		data[f.Path] = jsonnet.MakeContents(string(f.Content))
	}

	parts := make([]string, len(files))
	for i, f := range files {
		// Use double-quoted strings and escape \ and " as defense-in-depth,
		// even though safePath already rejects those characters.
		escaped := strings.ReplaceAll(f.Path, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		parts[i] = fmt.Sprintf(`(import "%s")`, escaped)
	}
	snippet := strings.Join(parts, " + ")

	vm := jsonnet.MakeVM()
	vm.MaxStack = 100
	vm.Importer(&jsonnet.MemoryImporter{Data: data})

	result, err := vm.EvaluateAnonymousSnippet("<gantry-merge>", snippet)
	if err != nil {
		return nil, fmt.Errorf("jsonnet evaluation: %w", err)
	}

	if len(result) > maxOutputBytes {
		return nil, fmt.Errorf("merged output exceeds maximum allowed size (%d bytes)", maxOutputBytes)
	}

	return []byte(strings.TrimSuffix(result, "\n")), nil
}
