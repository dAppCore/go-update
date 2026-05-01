package updater

import (
	"context"
	"net/http" // Note: AX-6 - structural HTTP transport boundary for update discovery responses.
	"net/url"

	"dappco.re/go"
)

// GenericUpdateInfo holds the information from a latest.json file.
// This file is expected to be at the root of a generic HTTP update server.
type GenericUpdateInfo struct {
	Version string `json:"version"` // The version number of the update.
	URL     string `json:"url"`     // The URL to download the update from.
}

// GetLatestUpdateFromURL fetches and parses a latest.json file from a base URL.
// The server at the baseURL should host a 'latest.json' file that contains
// the version and download URL for the latest update.
//
// Example of latest.json:
//
//	{
//	  "version": "1.2.3",
//	  "url": "https://your-server.com/path/to/release-asset"
//	}
func GetLatestUpdateFromURL(baseURL string) core.Result {
	u, err := url.Parse(baseURL)
	if err != nil {
		return core.Fail(core.E("GetLatestUpdateFromURL", "invalid base URL", err))
	}

	// Append latest.json to the path
	u.Path = core.Concat(core.TrimSuffix(u.Path, "/"), "/latest.json")

	request := newAgentRequest(context.Background(), "GET", u.String())
	if !request.OK {
		return core.Fail(core.E("GetLatestUpdateFromURL", "failed to create update check request", core.NewError(request.Error())))
	}

	resp, err := NewHTTPClient().Do(request.Value.(*http.Request))
	if err != nil {
		return core.Fail(core.E("GetLatestUpdateFromURL", "failed to fetch latest.json", err))
	}

	if resp.StatusCode != http.StatusOK {
		closeResponseBody(resp.Body)
		return core.Fail(core.E("GetLatestUpdateFromURL", core.Sprintf("failed to fetch latest.json: status code %d", resp.StatusCode), nil))
	}

	body := core.ReadAll(resp.Body)
	if !body.OK {
		if readErr, ok := body.Value.(error); ok {
			return core.Fail(core.E("GetLatestUpdateFromURL", "failed to read latest.json", readErr))
		}
		return core.Fail(core.E("GetLatestUpdateFromURL", "failed to read latest.json", nil))
	}

	var info GenericUpdateInfo
	// AX-6: latest.json is an HTTP response body boundary; decode through Core JSON.
	if result := core.JSONUnmarshal([]byte(body.Value.(string)), &info); !result.OK {
		if parseErr, ok := result.Value.(error); ok {
			return core.Fail(core.E("GetLatestUpdateFromURL", "failed to parse latest.json", parseErr))
		}
		return core.Fail(core.E("GetLatestUpdateFromURL", "failed to parse latest.json", nil))
	}

	if info.Version == "" || info.URL == "" {
		return core.Fail(core.E("GetLatestUpdateFromURL", "invalid latest.json content: version or url is missing", nil))
	}

	return core.Ok(&info)
}
