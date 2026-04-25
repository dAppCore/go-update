package updater

import (
	"context"
	"net/http" // Note: AX-6 - structural HTTP transport boundary for update client/request types.
	"time"

	"dappco.re/go/core"
)

const defaultHTTPTimeout = 30 * time.Second

var NewHTTPClient = func() *http.Client {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func newAgentRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", updaterUserAgent())
	return req, nil
}

func updaterUserAgent() string {
	version := formatVersionForDisplay(Version, true)
	if version == "" {
		version = "unknown"
	}
	return core.Sprintf("agent-go-update/%s", version)
}
