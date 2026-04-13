---
name: Gantry Security Architecture
description: Core security architecture and threat model for the Gantry devcontainer merge tool
type: project
---

Gantry is a Go CLI tool that clones git repositories into memory and merges devcontainer.json files using go-jsonnet. Key security-relevant facts:

- Input pipeline: user-supplied git URLs + file paths -> `git.Clone` (in-memory) -> `git.GetFiles` -> `merge.Merge` (Jsonnet evaluation)
- `merge.Merge` embeds user-controlled `f.Path` values directly into a Jsonnet snippet string without sanitization
- `MemoryImporter` restricts file access to the pre-loaded in-memory map, so the Jsonnet VM cannot reach the real filesystem IF paths are clean
- However, crafted paths with single-quote characters break the Jsonnet snippet string and allow injection into the Jsonnet program itself
- go-jsonnet stdlib (`std.*`) is available by default; no `MaxStack` or fuel limit is set, making resource exhaustion possible
- go-git/v6 is an alpha release (v6.0.0-alpha.1), which carries supply-chain/stability risk
- google/go-jsonnet is listed as an indirect dependency, suggesting it is not yet wired into the module's direct import graph at go.mod level (though it is used directly in merge.go)

**Why:** Security review of the merge implementation PR.
**How to apply:** Use this as baseline threat model for all future reviews of merge.go and git.go.
