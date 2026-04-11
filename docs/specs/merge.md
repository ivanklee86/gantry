# Jsonnet

Let's support using the `go-jsonnet` module (https://pkg.go.dev/github.com/google/go-jsonnet?utm_source=godoc#section-documentation) to "merge" devcontainer.json files together.  
- We should support both json files and jsonnet files.
- We should be able to evaluate 2...n files.

Integration tests should use actual, realistic examples (inc. using base devcontainer file as base file and jsonnet standard libray in second, jsonnet file).

Assume that files have the format of the `FileContent` struct from the go package.
