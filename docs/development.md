# Development

Run Go commands from `go/`.

```bash
GOWORK=off GOPROXY=direct GOSUMDB=off go build ./...
GOWORK=off GOPROXY=direct GOSUMDB=off go vet ./...
GOWORK=off GOPROXY=direct GOSUMDB=off go test -count=1 -short ./...
```

Run the v0.9.0 audit from the repository root.

```bash
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

