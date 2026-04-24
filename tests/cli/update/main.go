// AX-10 CLI driver for go-update. It exercises the public update flows without
// touching the running binary or calling external services.
//
//	task -d tests/cli/update
//	go run ./tests/cli/update
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"

	updater "dappco.re/go/update"
)

type githubClient struct{}

func (githubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return []string{"https://github.com/example/update.git"}, nil
}

func (githubClient) GetLatestRelease(ctx context.Context, owner, repo, channel string) (*updater.Release, error) {
	if owner != "example" || repo != "update" || channel != "stable" {
		return nil, fmt.Errorf("unexpected GitHub request: %s/%s channel %s", owner, repo, channel)
	}

	return &updater.Release{
		TagName: "v1.1.0",
		Assets: []updater.ReleaseAsset{
			{
				Name:        fmt.Sprintf("go-update-%s-%s", runtime.GOOS, runtime.GOARCH),
				DownloadURL: "https://updates.example.com/go-update",
			},
		},
	}, nil
}

func (githubClient) GetReleaseByPullRequest(ctx context.Context, owner, repo string, prNumber int) (*updater.Release, error) {
	return nil, nil
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
	updater.DoUpdate = func(url string) error {
		appliedURLs = append(appliedURLs, url)
		return nil
	}

	if err := runGitHubUpdate(); err != nil {
		fail(1, err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintln(w, `{"version":"1.2.0","url":"https://updates.example.com/http"}`)
	}))
	defer server.Close()

	if err := runHTTPUpdate(server.URL); err != nil {
		fail(2, err)
	}

	if len(appliedURLs) != 2 {
		fail(3, fmt.Errorf("expected 2 update applications, got %d", len(appliedURLs)))
	}
	if appliedURLs[0] != "https://updates.example.com/go-update" {
		fail(4, fmt.Errorf("unexpected GitHub update URL %q", appliedURLs[0]))
	}
	if appliedURLs[1] != "https://updates.example.com/http" {
		fail(5, fmt.Errorf("unexpected HTTP update URL %q", appliedURLs[1]))
	}
}

func runGitHubUpdate() error {
	service, err := updater.NewUpdateService(updater.UpdateServiceConfig{
		RepoURL:        "https://github.com/example/update",
		CheckOnStartup: updater.CheckAndUpdateOnStartup,
	})
	if err != nil {
		return err
	}
	return service.Start()
}

func runHTTPUpdate(baseURL string) error {
	service, err := updater.NewUpdateService(updater.UpdateServiceConfig{
		RepoURL:        baseURL,
		CheckOnStartup: updater.CheckAndUpdateOnStartup,
	})
	if err != nil {
		return err
	}
	return service.Start()
}

func fail(code int, err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(code)
}
