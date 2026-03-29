package updater

import (
	"context"
	"fmt"
	"net/http"
	"time"
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
	return fmt.Sprintf("agent-go-update/%s", version)
}
