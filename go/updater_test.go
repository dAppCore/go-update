package updater

import (
	"net/http"
	"net/http/httptest"

	. "dappco.re/go"
)

func TestUpdater_CheckForNewerVersion_Result(t *T) {
	originalVersion := Version
	originalNewGithubClient := NewGithubClient
	defer func() {
		Version = originalVersion
		NewGithubClient = originalNewGithubClient
	}()
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return updaterTestClient{release: &Release{TagName: "v1.1.0"}}
	}

	result := CheckForNewerVersion("core", "update", "stable", true)

	AssertTrue(t, result.OK)
	AssertTrue(t, result.Value.(versionCheck).updateAvailable)
}

// TestUpdater_VersionCheckResult_Good proves a versionCheck value satisfies
// VersionCheckResult and exposes its release + availability — the reuse path
// an external caller takes via check.Value.(updater.VersionCheckResult),
// without ever naming the unexported versionCheck type.
func TestUpdater_VersionCheckResult_Good(t *T) {
	var result VersionCheckResult = versionCheck{release: &Release{TagName: "v1.2.0"}, updateAvailable: true}

	AssertEqual(t, "v1.2.0", result.Release().TagName)
	AssertTrue(t, result.Available())
}

// TestUpdater_VersionCheckResult_Bad proves the zero-value case (no release
// found for the channel): Release() is nil, Available() is false.
func TestUpdater_VersionCheckResult_Bad(t *T) {
	var result VersionCheckResult = versionCheck{}

	AssertNil(t, result.Release())
	AssertFalse(t, result.Available())
}

// TestUpdater_VersionCheckResult_Ugly exercises the real end-to-end reuse
// case: CheckForNewerVersion's core.Result payload type-asserts to
// VersionCheckResult and yields the fetched release's assets, which is
// exactly what an external asset-selection routine needs and could not reach
// before this interface existed.
func TestUpdater_VersionCheckResult_Ugly(t *T) {
	originalVersion := Version
	originalNewGithubClient := NewGithubClient
	defer func() {
		Version = originalVersion
		NewGithubClient = originalNewGithubClient
	}()
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return updaterTestClient{release: &Release{
			TagName: "v1.1.0",
			Assets:  []ReleaseAsset{{Name: "linux-x86_64-lem-cpu-v1.1.0.zip"}},
		}}
	}

	check := CheckForNewerVersion("core", "update", "stable", true)

	AssertTrue(t, check.OK)
	vc := check.Value.(VersionCheckResult)
	AssertTrue(t, vc.Available())
	AssertEqual(t, "linux-x86_64-lem-cpu-v1.1.0.zip", vc.Release().Assets[0].Name)
}

// TestUpdater_DoUpdate_Good proves DoUpdate's URL-fetch contract is unchanged
// by the DoUpdateFromReader refactor: it still downloads the URL and hands
// the body onward — verified here by stubbing DoUpdateFromReader (the new
// seam) rather than exercising the real selfupdate.Apply, which would try to
// replace the test binary itself.
func TestUpdater_DoUpdate_Good(t *T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("binary-bytes"))
	}))
	defer server.Close()

	originalFromReader := DoUpdateFromReader
	defer func() { DoUpdateFromReader = originalFromReader }()
	var received string
	DoUpdateFromReader = func(r Reader) Result {
		body := ReadAll(r)
		if body.OK {
			received = body.Value.(string)
		}
		return Ok(nil)
	}

	result := DoUpdate(server.URL)

	AssertTrue(t, result.OK)
	AssertEqual(t, "binary-bytes", received)
}

// TestUpdater_DoUpdate_Bad proves a non-200 download response is rejected
// before ever reaching DoUpdateFromReader.
func TestUpdater_DoUpdate_Bad(t *T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalFromReader := DoUpdateFromReader
	defer func() { DoUpdateFromReader = originalFromReader }()
	called := false
	DoUpdateFromReader = func(r Reader) Result {
		called = true
		return Ok(nil)
	}

	result := DoUpdate(server.URL)

	AssertFalse(t, result.OK)
	AssertFalse(t, called)
}

// erroringReader fails on the first Read — used to drive DoUpdateFromReader's
// real error path (selfupdate.Apply reads the whole update into memory before
// touching the filesystem, so a reader that fails immediately never risks the
// test binary).
type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, E("erroringReader", "boom", nil) }

// TestUpdater_DoUpdateFromReader_Ugly exercises the REAL closure (not
// stubbed): a reader that fails mid-read makes selfupdate.Apply error out
// during its own ReadAll, before any file is opened, so this is safe to run
// against the actual function.
func TestUpdater_DoUpdateFromReader_Ugly(t *T) {
	result := DoUpdateFromReader(erroringReader{})

	AssertFalse(t, result.OK)
}

type updaterTestClient struct {
	release *Release
}

func (c updaterTestClient) GetPublicRepos(ctx Context, userOrOrg string) Result {
	return Ok([]string{"https://github.com/core/update.git"})
}

func (c updaterTestClient) GetLatestRelease(ctx Context, owner, repo, channel string) Result {
	return Ok(c.release)
}

func (c updaterTestClient) GetReleaseByPullRequest(ctx Context, owner, repo string, prNumber int) Result {
	return Ok(c.release)
}

func (c updaterTestClient) GetReleaseByTag(ctx Context, owner, repo, tag string) Result {
	return Ok(c.release)
}
