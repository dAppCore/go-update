module dappco.re/go/update

go 1.26.0

require (
	dappco.re/go v0.10.4
	github.com/minio/selfupdate v0.6.0 // Note: in-place binary self-update and rollback support; no core equivalent
	github.com/spf13/cobra v1.10.2 // Note: CLI command and flag wiring; no core equivalent
	golang.org/x/mod v0.34.0 // Note: semantic version comparison for update checks; no core equivalent
	golang.org/x/oauth2 v0.36.0 // Note: OAuth2-backed GitHub API HTTP client; no core equivalent
)

require (
	aead.dev/minisign v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
