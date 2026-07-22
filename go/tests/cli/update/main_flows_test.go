package main

import (
	"net/http"
	"net/http/httptest"

	. "dappco.re/go"
	updater "dappco.re/go/update"
)

// cliFlowsFixture pins the updater package seams the CLI driver depends on and
// returns the recorder slice for applied URLs plus a restore func.
func cliFlowsFixture() (*[]string, func()) {
	origVersion := updater.Version
	origDoUpdate := updater.DoUpdate
	origNewGithubClient := updater.NewGithubClient

	updater.Version = "1.0.0"
	updater.NewGithubClient = func() updater.GithubClient { return githubClient{} }

	applied := &[]string{}
	updater.DoUpdate = func(url string) Result {
		*applied = append(*applied, url)
		return Ok(nil)
	}

	return applied, func() {
		updater.Version = origVersion
		updater.DoUpdate = origDoUpdate
		updater.NewGithubClient = origNewGithubClient
	}
}

func TestMainFlows_RunGitHubUpdate_Good(t *T) {
	applied, restore := cliFlowsFixture()
	defer restore()

	result := runGitHubUpdate()

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/go-update"}, *applied)
}

func TestMainFlows_RunGitHubUpdate_Bad(t *T) {
	// An unparsable repo URL fails before any check happens.
	applied, restore := cliFlowsFixture()
	defer restore()

	result := updater.NewUpdateService(updater.UpdateServiceConfig{
		RepoURL:        "https://github.com/only-owner",
		CheckOnStartup: updater.CheckAndUpdateOnStartup,
	})

	AssertFalse(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestMainFlows_RunHTTPUpdate_Good(t *T) {
	applied, restore := cliFlowsFixture()
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			http.NotFound(w, r)
			return
		}
		write := WriteString(w, `{"version":"1.2.0","url":"https://updates.example.com/http"}`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	result := runHTTPUpdate(server.URL)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/http"}, *applied)
}

func TestMainFlows_RunHTTPUpdate_Bad(t *T) {
	// A server returning a 404 for latest.json fails the HTTP update.
	applied, restore := cliFlowsFixture()
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	result := runHTTPUpdate(server.URL)

	AssertFalse(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestMainFlows_RunHTTPUpdate_Ugly(t *T) {
	// Current version newer than the offered version: OK, nothing applied.
	applied, restore := cliFlowsFixture()
	defer restore()
	updater.Version = "5.0.0"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			http.NotFound(w, r)
			return
		}
		write := WriteString(w, `{"version":"1.2.0","url":"https://updates.example.com/old"}`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	result := runHTTPUpdate(server.URL)

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}
