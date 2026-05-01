package updater

import . "dappco.re/go"

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
