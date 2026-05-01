package main

import (
	. "dappco.re/go"
	updater "dappco.re/go/update"
)

func TestMain_Client_GetPublicRepos_Good(t *T) {
	result := githubClient{}.GetPublicRepos(Background(), "example")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://github.com/example/update.git"}, result.Value.([]string))
}

func TestMain_Client_GetPublicRepos_Bad(t *T) {
	result := githubClient{}.GetPublicRepos(Background(), "")

	AssertTrue(t, result.OK)
	AssertLen(t, result.Value.([]string), 1)
	AssertContains(t, result.Value.([]string)[0], "example/update.git")
}

func TestMain_Client_GetPublicRepos_Ugly(t *T) {
	ctx, cancel := WithCancel(Background())
	cancel()

	result := githubClient{}.GetPublicRepos(ctx, "ignored")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://github.com/example/update.git"}, result.Value.([]string))
}

func TestMain_Client_GetLatestRelease_Good(t *T) {
	result := githubClient{}.GetLatestRelease(Background(), "example", "update", "stable")

	AssertTrue(t, result.OK)
	release := result.Value.(*updater.Release)
	AssertEqual(t, "v1.1.0", release.TagName)
	AssertContains(t, release.Assets[0].Name, OS())
	AssertContains(t, release.Assets[0].Name, Arch())
}

func TestMain_Client_GetLatestRelease_Bad(t *T) {
	result := githubClient{}.GetLatestRelease(Background(), "wrong", "update", "stable")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unexpected GitHub request")
}

func TestMain_Client_GetLatestRelease_Ugly(t *T) {
	result := githubClient{}.GetLatestRelease(Background(), "example", "update", " beta ")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "channel  beta ")
}

func TestMain_Client_GetReleaseByPullRequest_Good(t *T) {
	result := githubClient{}.GetReleaseByPullRequest(Background(), "example", "update", 123)

	AssertTrue(t, result.OK)
	AssertNil(t, result.Value.(*updater.Release))
}

func TestMain_Client_GetReleaseByPullRequest_Bad(t *T) {
	result := githubClient{}.GetReleaseByPullRequest(Background(), "wrong", "update", -1)

	AssertTrue(t, result.OK)
	AssertNil(t, result.Value.(*updater.Release))
}

func TestMain_Client_GetReleaseByPullRequest_Ugly(t *T) {
	ctx, cancel := WithCancel(Background())
	cancel()

	result := githubClient{}.GetReleaseByPullRequest(ctx, "", "", 0)

	AssertTrue(t, result.OK)
	AssertNil(t, result.Value.(*updater.Release))
}
