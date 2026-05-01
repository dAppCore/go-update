package updater

import (
	"context"
	"net/http" // Note: AX-6 - structural HTTP transport boundary for update client/request types.
	"time"

	"dappco.re/go"
)

const defaultHTTPTimeout = 30 * time.Second

var NewHTTPClient = func() *http.Client {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func newAgentRequest(ctx context.Context, method, url string) core.Result {
	r := core.NewHTTPRequestContext(ctx, method, url, nil)
	if !r.OK {
		return r
	}
	req := r.Value.(*http.Request)
	req.Header.Set("User-Agent", updaterUserAgent())
	return core.Ok(req)
}

func updaterUserAgent() string {
	version := formatVersionForDisplay(Version, true)
	if version == "" {
		version = "unknown"
	}
	return core.Sprintf("agent-go-update/%s", version)
}
