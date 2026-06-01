package updater

import (
	. "dappco.re/go"
	"github.com/spf13/cobra"
)

// cmdRunFixture snapshots the mutable command state and the package seams that
// runUpdate touches, restoring them when the returned func is deferred.
func cmdRunFixture() func() {
	origChannel := updateChannel
	origForce := updateForce
	origCheck := updateCheck
	origWatchPID := updateWatchPID
	origVersion := Version
	origNewGithubClient := NewGithubClient
	origDoUpdate := DoUpdate
	origSpawnWatcher := spawnWatcher

	// Default: a watcher that never actually forks a process.
	spawnWatcher = func() Result { return Ok(nil) }

	return func() {
		updateChannel = origChannel
		updateForce = origForce
		updateCheck = origCheck
		updateWatchPID = origWatchPID
		Version = origVersion
		NewGithubClient = origNewGithubClient
		DoUpdate = origDoUpdate
		spawnWatcher = origSpawnWatcher
	}
}

func TestCmdRun_RunUpdate_Good(t *T) {
	defer cmdRunFixture()()
	updateChannel = "stable"
	updateCheck = false
	updateForce = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{
			TagName: "v1.1.0",
			Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/app"}},
		}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/app"}, applied)
}

func TestCmdRun_RunUpdate_Bad(t *T) {
	defer cmdRunFixture()()
	updateChannel = "stable"
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latestErr: "github unreachable"}
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to check for updates")
}

func TestCmdRun_RunUpdate_Ugly(t *T) {
	// No release in channel: OK result, nothing applied.
	defer cmdRunFixture()()
	updateChannel = "beta"
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: nil}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertLen(t, applied, 0)
}

func TestCmdRun_RunUpdate_AlreadyLatest(t *T) {
	defer cmdRunFixture()()
	updateChannel = "stable"
	updateForce = false
	Version = "2.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{TagName: "v1.9.0"}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertLen(t, applied, 0)
}

func TestCmdRun_RunUpdate_CheckOnly(t *T) {
	// --check reports the available update but never downloads.
	defer cmdRunFixture()()
	updateChannel = "stable"
	updateCheck = true
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{TagName: "v1.2.0"}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertLen(t, applied, 0)
}

func TestCmdRun_RunUpdate_DownloadURLError(t *T) {
	// Update available but no matching asset: GetDownloadURL fails.
	defer cmdRunFixture()()
	updateChannel = "stable"
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{TagName: "v1.1.0"}}
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to get download URL")
}

func TestCmdRun_RunUpdate_ApplyError(t *T) {
	// DoUpdate fails (e.g. failed rollback): runUpdate wraps the error.
	defer cmdRunFixture()()
	updateChannel = "stable"
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{
			TagName: "v1.1.0",
			Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/app"}},
		}}
	}
	DoUpdate = func(url string) Result {
		return Fail(E("DoUpdate", "update failed", nil))
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), failedToApplyUpdate)
}

func TestCmdRun_RunUpdate_ForceUpToDate(t *T) {
	// Already on latest but --force still applies the update.
	defer cmdRunFixture()()
	updateChannel = "stable"
	updateForce = true
	Version = "2.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{
			TagName: "v2.0.0",
			Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/force"}},
		}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/force"}, applied)
}

func TestCmdRun_HandleDevUpdate_Good(t *T) {
	// dev channel resolves a beta release and applies it.
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
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/dev"}, applied)
}

func TestCmdRun_HandleDevUpdate_Bad(t *T) {
	// dev channel with a failing release lookup falls back to the dev-tag
	// download URL and applies that instead.
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latestErr: "no beta channel"}
	}
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

func TestCmdRun_HandleDevUpdate_Ugly(t *T) {
	// dev channel in --check mode reports but never applies.
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = true
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{TagName: "v1.1.0-beta.1"}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertLen(t, applied, 0)
}

func TestCmdRun_HandleDevTagUpdate_Good(t *T) {
	// A nil beta release routes to the dev-tag fallback download.
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: nil}
	}
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

func TestCmdRun_HandleDevTagUpdate_Bad(t *T) {
	// dev-tag fallback in --check mode reports but never applies.
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = true
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: nil}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertTrue(t, result.OK)
	AssertLen(t, applied, 0)
}

func TestCmdRun_HandleDevTagUpdate_Ugly(t *T) {
	// dev-tag fallback where DoUpdate fails surfaces the apply error.
	defer cmdRunFixture()()
	updateChannel = "dev"
	updateCheck = false
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: nil}
	}
	DoUpdate = func(url string) Result {
		return Fail(E("DoUpdate", "download failed", nil))
	}

	result := runUpdate(&cobra.Command{}, nil)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), failedToApplyUpdate)
}

func TestCmdRun_UpdateCommandError_Good(t *T) {
	result := updateCommandError(NewError("disk full"), "failed to apply update")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to apply update")
}

func TestCmdRun_UpdateCommandError_Bad(t *T) {
	// A nil cause is dropped by core.Wrap, so the wrapper message is lost and
	// the result surfaces as a generic failure. In production every caller
	// passes a non-nil core.NewError(...), so this path is defensive only.
	result := updateCommandError(nil, "failed to get download URL")

	AssertFalse(t, result.OK)
	AssertNotEqual(t, "", result.Error())
}

func TestCmdRun_UpdateCommandError_Ugly(t *T) {
	result := updateCommandError(NewError("boom"), "")

	AssertFalse(t, result.OK)
	AssertNotEqual(t, "", result.Error())
}
