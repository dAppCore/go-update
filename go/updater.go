package updater

import (
	"context"
	"net/http"

	core "dappco.re/go"
	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"
)

const currentVersionUpToDateFormat = "Current version %s is up-to-date with latest release %s.\n"

// Version holds the current version of the application.
// It is set at build time via ldflags or fallback to the version in package.json.
var Version = PkgVersion

// NewGithubClient is a variable that holds a function to create a new GithubClient.
// This can be replaced in tests to inject a mock client.
//
// Example:
//
//	updater.NewGithubClient = func() updater.GithubClient {
//		return &mockClient{} // or your mock implementation
//	}
var NewGithubClient = func() GithubClient {
	return &githubClient{}
}

type versionCheck struct {
	release         *Release
	updateAvailable bool
}

// DoUpdate is a variable that holds the function to perform the actual update.
// This can be replaced in tests to prevent actual updates.
var DoUpdate = func(url string) core.Result {
	client := NewHTTPClient()
	request := newAgentRequest(context.Background(), "GET", url)
	if !request.OK {
		return core.Fail(core.E("DoUpdate", "failed to create update request", core.NewError(request.Error())))
	}

	resp, err := client.Do(request.Value.(*http.Request))
	if err != nil {
		return core.Fail(core.E("DoUpdate", "failed to download update", err))
	}
	defer closeResponseBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return core.Fail(core.E("DoUpdate", core.Sprintf("failed to download update: %s", resp.Status), nil))
	}

	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return core.Fail(core.E("DoUpdate", "failed to rollback from failed update", rerr))
		}
		return core.Fail(core.E("DoUpdate", "update failed", err))
	}
	return core.Ok(nil)
}

// CheckForNewerVersion checks if a newer version of the application is available on GitHub.
// It fetches the latest release for the given owner, repository, and channel, and compares its tag
// with the current application version.
var CheckForNewerVersion = func(owner, repo, channel string, forceSemVerPrefix bool) core.Result {
	client := NewGithubClient()
	ctx := context.Background()

	result := client.GetLatestRelease(ctx, owner, repo, channel)
	if !result.OK {
		return core.Fail(core.E("CheckForNewerVersion", "error fetching latest release", core.NewError(result.Error())))
	}
	release := result.Value.(*Release)

	if release == nil {
		return core.Ok(versionCheck{}) // No release found
	}

	// Always normalize to 'v' prefix for semver comparison
	vCurrent := formatVersionForComparison(Version)
	vLatest := formatVersionForComparison(release.TagName)

	if semver.Compare(vCurrent, vLatest) >= 0 {
		return core.Ok(versionCheck{release: release}) // Current version is up-to-date or newer
	}

	return core.Ok(versionCheck{release: release, updateAvailable: true}) // A newer version is available
}

// CheckForUpdates checks for new updates on GitHub and applies them if a newer version is found.
// It uses the provided owner, repository, and channel to find the latest release.
var CheckForUpdates = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) core.Result {
	check := CheckForNewerVersion(owner, repo, channel, forceSemVerPrefix)
	if !check.OK {
		return check
	}
	state := check.Value.(versionCheck)

	if !state.updateAvailable {
		if state.release != nil {
			core.Print(nil, currentVersionUpToDateFormat,
				formatVersionForDisplay(Version, forceSemVerPrefix),
				formatVersionForDisplay(state.release.TagName, forceSemVerPrefix))
		} else {
			core.Println("No releases found.")
		}
		return core.Ok(nil)
	}

	core.Print(nil, "Newer version %s found (current: %s). Applying update...",
		formatVersionForDisplay(state.release.TagName, forceSemVerPrefix),
		formatVersionForDisplay(Version, forceSemVerPrefix))

	downloadURL := GetDownloadURL(state.release, releaseURLFormat)
	if !downloadURL.OK {
		return core.Fail(core.E("CheckForUpdates", "error getting download URL", core.NewError(downloadURL.Error())))
	}

	return DoUpdate(downloadURL.Value.(string))
}

// CheckOnly checks for new updates on GitHub without applying them.
// It prints a message indicating if a new release is available.
var CheckOnly = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) core.Result {
	check := CheckForNewerVersion(owner, repo, channel, forceSemVerPrefix)
	if !check.OK {
		return check
	}
	state := check.Value.(versionCheck)

	if !state.updateAvailable {
		if state.release != nil {
			core.Print(nil, currentVersionUpToDateFormat,
				formatVersionForDisplay(Version, forceSemVerPrefix),
				formatVersionForDisplay(state.release.TagName, forceSemVerPrefix))
		} else {
			core.Println("No new release found.")
		}
		return core.Ok(nil)
	}

	core.Print(nil, "New release found: %s (current version: %s)",
		formatVersionForDisplay(state.release.TagName, forceSemVerPrefix),
		formatVersionForDisplay(Version, forceSemVerPrefix))
	return core.Ok(nil)
}

// CheckForUpdatesByTag checks for and applies updates from GitHub based on the channel
// determined by the current application's version tag (e.g., 'stable' or 'prerelease').
var CheckForUpdatesByTag = func(owner, repo string) core.Result {
	channel := determineChannel(Version, semver.Prerelease(formatVersionForComparison(Version)) != "")
	return CheckForUpdates(owner, repo, channel, true, "")
}

// CheckOnlyByTag checks for updates from GitHub based on the channel determined by the
// current version tag, without applying them.
var CheckOnlyByTag = func(owner, repo string) core.Result {
	channel := determineChannel(Version, semver.Prerelease(formatVersionForComparison(Version)) != "")
	return CheckOnly(owner, repo, channel, true, "")
}

// CheckForUpdatesByPullRequest finds a release associated with a specific pull request number
// on GitHub and applies the update.
var CheckForUpdatesByPullRequest = func(owner, repo string, prNumber int, releaseURLFormat string) core.Result {
	client := NewGithubClient()
	ctx := context.Background()

	result := client.GetReleaseByPullRequest(ctx, owner, repo, prNumber)
	if !result.OK {
		return core.Fail(core.E("CheckForUpdatesByPullRequest", "error fetching release for pull request", core.NewError(result.Error())))
	}
	release := result.Value.(*Release)

	if release == nil {
		core.Print(nil, "No release found for PR #%d.", prNumber)
		return core.Ok(nil)
	}

	core.Print(nil, "Release %s found for PR #%d. Applying update...", release.TagName, prNumber)

	downloadURL := GetDownloadURL(release, releaseURLFormat)
	if !downloadURL.OK {
		return core.Fail(core.E("CheckForUpdatesByPullRequest", "error getting download URL", core.NewError(downloadURL.Error())))
	}

	return DoUpdate(downloadURL.Value.(string))
}

// CheckForUpdatesHTTP checks for and applies updates from a generic HTTP endpoint.
// The endpoint is expected to provide update information in a structured format.
var CheckForUpdatesHTTP = func(baseURL string) core.Result {
	result := GetLatestUpdateFromURL(baseURL)
	if !result.OK {
		return result
	}
	info := result.Value.(*GenericUpdateInfo)

	vCurrent := formatVersionForComparison(Version)
	vLatest := formatVersionForComparison(info.Version)

	if semver.Compare(vCurrent, vLatest) >= 0 {
		core.Print(nil, currentVersionUpToDateFormat, Version, info.Version)
		return core.Ok(nil)
	}

	core.Print(nil, "Newer version %s found (current: %s). Applying update...", info.Version, Version)
	return DoUpdate(info.URL)
}

// CheckOnlyHTTP checks for updates from a generic HTTP endpoint without applying them.
// It prints a message if a new version is available.
var CheckOnlyHTTP = func(baseURL string) core.Result {
	result := GetLatestUpdateFromURL(baseURL)
	if !result.OK {
		return result
	}
	info := result.Value.(*GenericUpdateInfo)

	vCurrent := formatVersionForComparison(Version)
	vLatest := formatVersionForComparison(info.Version)

	if semver.Compare(vCurrent, vLatest) >= 0 {
		core.Print(nil, currentVersionUpToDateFormat, Version, info.Version)
		return core.Ok(nil)
	}

	core.Print(nil, "New release found: %s (current version: %s)", info.Version, Version)
	return core.Ok(nil)
}

// formatVersionForComparison ensures the version string has a 'v' prefix for semver comparison.
func formatVersionForComparison(version string) string {
	if version != "" && !core.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// formatVersionForDisplay ensures the version string has the correct 'v' prefix based on the forceSemVerPrefix flag.
func formatVersionForDisplay(version string, forceSemVerPrefix bool) string {
	hasV := core.HasPrefix(version, "v")
	if forceSemVerPrefix && !hasV {
		return "v" + version
	}
	if !forceSemVerPrefix && hasV {
		return core.TrimPrefix(version, "v")
	}
	return version
}

func closeResponseBody(body core.Closer) core.Result {
	if body == nil {
		return core.Ok(nil)
	}
	if err := body.Close(); err != nil {
		// Close errors are not actionable for update discovery/download response bodies.
		return core.Fail(err)
	}
	return core.Ok(nil)
}
