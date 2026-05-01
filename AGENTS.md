# AGENTS

This repository follows the core/go v0.9.0 consumer layout.

Go code lives under `go/`, with the repository root carrying `go.work` and
documentation. Local core dependencies are mounted through `external/` and
referenced by the workspace instead of `replace` directives.

