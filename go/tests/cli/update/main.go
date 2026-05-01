// AX-10 CLI driver for go-update. It exercises the public update flows without
// touching the running binary or calling external services.
//
//	task -d tests/cli/update
//	go run ./tests/cli/update
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"

	core "dappco.re/go"
	updater "dappco.re/go/update"
)

type githubClient struct{}

// Client exposes the fixture client method set for examples.
type Client = githubClient

func (githubClient) GetPublicRepos(ctx context.Context, userOrOrg string) core.Result {
	return core.Ok([]string{"https://github.com/example/update.git"})
}

func (githubClient) GetLatestRelease(ctx context.Context, owner, repo, channel string) core.Result {
	if owner != "example" || repo != "update" || channel != "stable" {
		return core.Fail(core.Errorf("unexpected GitHub request: %s/%s channel %s", owner, repo, channel))
	}

	return core.Ok(&updater.Release{
		TagName: "v1.1.0",
		Assets: []updater.ReleaseAsset{
			{
				Name:        core.Sprintf("go-update-%s-%s", runtime.GOOS, runtime.GOARCH),
				DownloadURL: "https://updates.example.com/go-update",
			},
		},
	})
}

func (githubClient) GetReleaseByPullRequest(ctx context.Context, owner, repo string, prNumber int) core.Result {
	return core.Ok((*updater.Release)(nil))
}

func main() {
	originalVersion := updater.Version
	originalDoUpdate := updater.DoUpdate
	originalNewGithubClient := updater.NewGithubClient
	defer func() {
		updater.Version = originalVersion
		updater.DoUpdate = originalDoUpdate
		updater.NewGithubClient = originalNewGithubClient
	}()

	updater.Version = "1.0.0"
	updater.NewGithubClient = func() updater.GithubClient {
		return githubClient{}
	}

	appliedURLs := make([]string, 0, 2)
	updater.DoUpdate = func(url string) core.Result {
		appliedURLs = append(appliedURLs, url)
		return core.Ok(nil)
	}

	if r := runGitHubUpdate(); !r.OK {
		fail(1, r.Error())
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			http.NotFound(w, r)
			return
		}
		write := core.WriteString(w, `{"version":"1.2.0","url":"https://updates.example.com/http"}`)
		if !write.OK {
			return
		}
	}))
	defer server.Close()

	if r := runHTTPUpdate(server.URL); !r.OK {
		fail(2, r.Error())
	}

	if len(appliedURLs) != 2 {
		fail(3, core.Sprintf("expected 2 update applications, got %d", len(appliedURLs)))
	}
	if appliedURLs[0] != "https://updates.example.com/go-update" {
		fail(4, core.Sprintf("unexpected GitHub update URL %q", appliedURLs[0]))
	}
	if appliedURLs[1] != "https://updates.example.com/http" {
		fail(5, core.Sprintf("unexpected HTTP update URL %q", appliedURLs[1]))
	}
}

func runGitHubUpdate() core.Result {
	result := updater.NewUpdateService(updater.UpdateServiceConfig{
		RepoURL:        "https://github.com/example/update",
		CheckOnStartup: updater.CheckAndUpdateOnStartup,
	})
	if !result.OK {
		return result
	}
	service := result.Value.(*updater.UpdateService)
	return service.Start()
}

func runHTTPUpdate(baseURL string) core.Result {
	result := updater.NewUpdateService(updater.UpdateServiceConfig{
		RepoURL:        baseURL,
		CheckOnStartup: updater.CheckAndUpdateOnStartup,
	})
	if !result.OK {
		return result
	}
	service := result.Value.(*updater.UpdateService)
	return service.Start()
}

func fail(code int, message string) {
	core.Print(core.Stderr(), "%s", message)
	core.Exit(code)
}
