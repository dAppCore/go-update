package updater

import (
	. "dappco.re/go"
	"github.com/spf13/cobra"
)

func TestCmdWatcherWarn_RunUpdate_WatcherFails(t *T) {
	// A failing watcher only emits a warning; the update still applies.
	defer cmdRunFixture()()
	updateChannel = "stable"
	updateCheck = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{
			TagName: "v1.1.0",
			Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/app"}},
		}}
	}
	spawnWatcher = func() Result { return Fail(E("spawnWatcher", "no executable", nil)) }
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/app"}, applied)
}

func TestCmdWatcherWarn_HandleDevUpdate_WatcherFails(t *T) {
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{
			TagName: "v1.1.0-beta.1",
			Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/dev"}},
		}}
	}
	spawnWatcher = func() Result { return Fail(E("spawnWatcher", "no executable", nil)) }
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/dev"}, applied)
}

func TestCmdWatcherWarn_HandleDevTagUpdate_WatcherFails(t *T) {
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: nil}
	}
	spawnWatcher = func() Result { return Fail(E("spawnWatcher", "no executable", nil)) }
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertLen(t, applied, 1)
	AssertContains(t, applied[0], "releases/download/dev/core-")
}
