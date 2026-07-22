package updater

import (
	"net/http"

	. "dappco.re/go"
)

// flowsTestClient is a configurable GithubClient stub for exercising the
// high-level update flows in updater.go without touching the network.
type flowsTestClient struct {
	latest    *Release
	latestErr string
	pr        *Release
	prErr     string
}

func (c flowsTestClient) GetPublicRepos(ctx Context, userOrOrg string) Result {
	return Ok([]string{})
}

func (c flowsTestClient) GetLatestRelease(ctx Context, owner, repo, channel string) Result {
	if c.latestErr != "" {
		return Fail(E("flowsTestClient.GetLatestRelease", c.latestErr, nil))
	}
	return Ok(c.latest)
}

func (c flowsTestClient) GetReleaseByPullRequest(ctx Context, owner, repo string, prNumber int) Result {
	if c.prErr != "" {
		return Fail(E("flowsTestClient.GetReleaseByPullRequest", c.prErr, nil))
	}
	return Ok(c.pr)
}

// GetReleaseByTag is unused by any updater.go flow (it is a primitive for
// callers that need an exact-tag lookup, e.g. a rolling "dev" release) but is
// still required to satisfy GithubClient.
func (c flowsTestClient) GetReleaseByTag(ctx Context, owner, repo, tag string) Result {
	return Ok((*Release)(nil))
}

// withFlowsClient swaps NewGithubClient for the test duration, returning a
// restore func the caller defers.
func withFlowsClient(client GithubClient) func() {
	original := NewGithubClient
	NewGithubClient = func() GithubClient { return client }
	return func() { NewGithubClient = original }
}

// withFlowsVersion pins Version for the test duration.
func withFlowsVersion(v string) func() {
	original := Version
	Version = v
	return func() { Version = original }
}

// captureDoUpdate replaces DoUpdate with a recorder, returning the slice of
// applied URLs and a restore func.
func captureDoUpdate(result Result) (*[]string, func()) {
	original := DoUpdate
	applied := &[]string{}
	DoUpdate = func(url string) Result {
		*applied = append(*applied, url)
		return result
	}
	return applied, func() { DoUpdate = original }
}

func TestUpdaterFlows_CheckForUpdates_Good(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{
		TagName: "v1.1.0",
		Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/app"}},
	}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdates("core", "update", "stable", true, "")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/app"}, *applied)
}

func TestUpdaterFlows_CheckForUpdates_Bad(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latestErr: "github unreachable"})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdates("core", "update", "stable", true, "")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error fetching latest release")
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckForUpdates_Ugly(t *T) {
	// Up-to-date: latest equals current, no update applied, OK result.
	defer withFlowsVersion("1.1.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{TagName: "v1.1.0"}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdates("core", "update", "stable", true, "")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckForUpdates_NoRelease(t *T) {
	// Nil release surfaces "No releases found." and an OK result.
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: nil})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdates("core", "update", "stable", true, "")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckForUpdates_DownloadURLError(t *T) {
	// Update available but the release carries no asset for this platform and
	// no URL template is provided, so GetDownloadURL fails before DoUpdate.
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{TagName: "v1.1.0"}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdates("core", "update", "stable", true, "")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error getting download URL")
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckForUpdates_DoUpdateFails(t *T) {
	// The rollback/apply failure path: DoUpdate returns Fail and the error
	// propagates out of CheckForUpdates.
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{TagName: "v1.1.0"}})()
	_, restore := captureDoUpdate(Fail(E("DoUpdate", "failed to rollback from failed update", nil)))
	defer restore()

	result := CheckForUpdates("core", "update", "stable", true, "https://updates.example.com/{tag}")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to rollback from failed update")
}

func TestUpdaterFlows_CheckOnly_Good(t *T) {
	// Update available: CheckOnly reports but never applies.
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{TagName: "v2.0.0"}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckOnly("core", "update", "stable", true, "")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckOnly_Bad(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latestErr: "rate limited"})()

	result := CheckOnly("core", "update", "stable", true, "")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error fetching latest release")
}

func TestUpdaterFlows_CheckOnly_Ugly(t *T) {
	// No release at all: still OK, prints "No new release found."
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: nil})()

	result := CheckOnly("core", "update", "stable", true, "")

	AssertTrue(t, result.OK)
}

func TestUpdaterFlows_CheckOnly_UpToDate(t *T) {
	defer withFlowsVersion("3.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{TagName: "v2.9.0"}})()

	result := CheckOnly("core", "update", "stable", true, "")

	AssertTrue(t, result.OK)
}

func TestUpdaterFlows_CheckForUpdatesByTag_Good(t *T) {
	// A stable current version routes to the stable channel and applies.
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{
		TagName: "v1.1.0",
		Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/tag"}},
	}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesByTag("core", "update")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/tag"}, *applied)
}

func TestUpdaterFlows_CheckForUpdatesByTag_Bad(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latestErr: "boom"})()

	result := CheckForUpdatesByTag("core", "update")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error fetching latest release")
}

func TestUpdaterFlows_CheckForUpdatesByTag_Ugly(t *T) {
	// A prerelease current version (alpha) routes to the alpha channel; the
	// stub returns a matching alpha release so the channel selection is exercised.
	defer withFlowsVersion("v1.0.0-alpha.1")()
	defer withFlowsClient(flowsTestClient{latest: &Release{
		TagName: "v1.0.0-alpha.2",
		Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/alpha"}},
	}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesByTag("core", "update")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/alpha"}, *applied)
}

func TestUpdaterFlows_CheckOnlyByTag_Good(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latest: &Release{TagName: "v1.2.0"}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckOnlyByTag("core", "update")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckOnlyByTag_Bad(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withFlowsClient(flowsTestClient{latestErr: "denied"})()

	result := CheckOnlyByTag("core", "update")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error fetching latest release")
}

func TestUpdaterFlows_CheckOnlyByTag_Ugly(t *T) {
	defer withFlowsVersion("v2.0.0-beta.1")()
	defer withFlowsClient(flowsTestClient{latest: nil})()

	result := CheckOnlyByTag("core", "update")

	AssertTrue(t, result.OK)
}

func TestUpdaterFlows_CheckForUpdatesByPullRequest_Good(t *T) {
	defer withFlowsClient(flowsTestClient{pr: &Release{
		TagName: "v1.2.0-alpha.pr.123",
		Assets:  []ReleaseAsset{{Name: Concat("app-", OS(), "-", Arch()), DownloadURL: "https://updates.example.com/pr"}},
	}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesByPullRequest("core", "update", 123, "")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/pr"}, *applied)
}

func TestUpdaterFlows_CheckForUpdatesByPullRequest_Bad(t *T) {
	defer withFlowsClient(flowsTestClient{prErr: "no such PR"})()

	result := CheckForUpdatesByPullRequest("core", "update", 999, "")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error fetching release for pull request")
}

func TestUpdaterFlows_CheckForUpdatesByPullRequest_Ugly(t *T) {
	// No release for the PR: OK, no download attempted.
	defer withFlowsClient(flowsTestClient{pr: nil})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesByPullRequest("core", "update", 123, "")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckForUpdatesByPullRequest_DownloadURLError(t *T) {
	// PR release found but no matching asset and no template: download fails.
	defer withFlowsClient(flowsTestClient{pr: &Release{TagName: "v1.2.0-alpha.pr.123"}})()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesByPullRequest("core", "update", 123, "")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "error getting download URL")
	AssertLen(t, *applied, 0)
}

// httpFlowsRoundTrip lets generic-HTTP flow tests intercept the update client
// without standing up a server, so latest.json content is fully controlled.
type httpFlowsRoundTrip func(*http.Request) (*http.Response, error)

func (f httpFlowsRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type httpFlowsReadCloser struct {
	Reader
}

func (httpFlowsReadCloser) Close() error { return nil }

func withHTTPFlowsClient(body string, status int) func() {
	original := NewHTTPClient
	NewHTTPClient = func() *http.Client {
		return &http.Client{Transport: httpFlowsRoundTrip(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Status:     Sprintf("%d %s", status, HTTPStatusText(status)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       httpFlowsReadCloser{Reader: NewReader(body)},
				Request:    req,
			}, nil
		})}
	}
	return func() { NewHTTPClient = original }
}

func TestUpdaterFlows_CheckForUpdatesHTTP_Good(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withHTTPFlowsClient(`{"version":"1.1.0","url":"https://updates.example.com/http"}`, http.StatusOK)()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesHTTP("https://updates.example.com")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://updates.example.com/http"}, *applied)
}

func TestUpdaterFlows_CheckForUpdatesHTTP_Bad(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withHTTPFlowsClient(``, http.StatusInternalServerError)()

	result := CheckForUpdatesHTTP("https://updates.example.com")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to fetch latest.json")
}

func TestUpdaterFlows_CheckForUpdatesHTTP_Ugly(t *T) {
	// Current version newer than offered: no update applied, OK result.
	defer withFlowsVersion("2.0.0")()
	defer withHTTPFlowsClient(`{"version":"1.0.0","url":"https://updates.example.com/old"}`, http.StatusOK)()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckForUpdatesHTTP("https://updates.example.com")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckOnlyHTTP_Good(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withHTTPFlowsClient(`{"version":"1.5.0","url":"https://updates.example.com/http"}`, http.StatusOK)()
	applied, restore := captureDoUpdate(Ok(nil))
	defer restore()

	result := CheckOnlyHTTP("https://updates.example.com")

	AssertTrue(t, result.OK)
	AssertLen(t, *applied, 0)
}

func TestUpdaterFlows_CheckOnlyHTTP_Bad(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withHTTPFlowsClient(`{"version":`, http.StatusOK)()

	result := CheckOnlyHTTP("https://updates.example.com")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to parse latest.json")
}

func TestUpdaterFlows_CheckOnlyHTTP_Ugly(t *T) {
	defer withFlowsVersion("1.0.0")()
	defer withHTTPFlowsClient(`{"version":"1.0.0","url":"https://updates.example.com/same"}`, http.StatusOK)()

	result := CheckOnlyHTTP("https://updates.example.com")

	AssertTrue(t, result.OK)
}

func TestUpdaterFlows_FormatVersionForComparison_Good(t *T) {
	AssertEqual(t, "v1.2.3", formatVersionForComparison("1.2.3"))
}

func TestUpdaterFlows_FormatVersionForComparison_Bad(t *T) {
	// Empty version is left empty (no spurious 'v' prefix).
	AssertEqual(t, "", formatVersionForComparison(""))
}

func TestUpdaterFlows_FormatVersionForComparison_Ugly(t *T) {
	// Already-prefixed version is untouched.
	AssertEqual(t, "v2.0.0", formatVersionForComparison("v2.0.0"))
}

func TestUpdaterFlows_FormatVersionForDisplay_Good(t *T) {
	AssertEqual(t, "v1.2.3", formatVersionForDisplay("1.2.3", true))
}

func TestUpdaterFlows_FormatVersionForDisplay_Bad(t *T) {
	// forceSemVerPrefix false strips an existing 'v'.
	AssertEqual(t, "1.2.3", formatVersionForDisplay("v1.2.3", false))
}

func TestUpdaterFlows_FormatVersionForDisplay_Ugly(t *T) {
	// Already in the requested shape: returned unchanged either way.
	AssertEqual(t, "v1.2.3", formatVersionForDisplay("v1.2.3", true))
	AssertEqual(t, "1.2.3", formatVersionForDisplay("1.2.3", false))
}

func TestUpdaterFlows_CloseResponseBody_Good(t *T) {
	result := closeResponseBody(httpFlowsReadCloser{Reader: NewReader("body")})

	AssertTrue(t, result.OK)
}

func TestUpdaterFlows_CloseResponseBody_Bad(t *T) {
	// A nil body is a no-op success.
	result := closeResponseBody(nil)

	AssertTrue(t, result.OK)
}

func TestUpdaterFlows_CloseResponseBody_Ugly(t *T) {
	// A body whose Close returns an error surfaces as Fail.
	result := closeResponseBody(closeErrReadCloser{})

	AssertFalse(t, result.OK)
}

type closeErrReadCloser struct{}

func (closeErrReadCloser) Read(p []byte) (int, error) { return 0, nil }
func (closeErrReadCloser) Close() error               { return E("closeErrReadCloser", "close failed", nil) }
