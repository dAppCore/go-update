package main

import (
	. "dappco.re/go"
)

func TestAX7_Client_GetPublicRepos_Good(t *T) {
	repos, err := githubClient{}.GetPublicRepos(Background(), "example")

	AssertNoError(t, err)
	AssertEqual(t, []string{"https://github.com/example/update.git"}, repos)
}

func TestAX7_Client_GetPublicRepos_Bad(t *T) {
	repos, err := githubClient{}.GetPublicRepos(Background(), "")

	AssertNoError(t, err)
	AssertLen(t, repos, 1)
	AssertContains(t, repos[0], "example/update.git")
}

func TestAX7_Client_GetPublicRepos_Ugly(t *T) {
	ctx, cancel := WithCancel(Background())
	cancel()

	repos, err := githubClient{}.GetPublicRepos(ctx, "ignored")

	AssertNoError(t, err)
	AssertEqual(t, []string{"https://github.com/example/update.git"}, repos)
}

func TestAX7_Client_GetLatestRelease_Good(t *T) {
	release, err := githubClient{}.GetLatestRelease(Background(), "example", "update", "stable")

	AssertNoError(t, err)
	AssertNotNil(t, release)
	AssertEqual(t, "v1.1.0", release.TagName)
	AssertContains(t, release.Assets[0].Name, OS())
	AssertContains(t, release.Assets[0].Name, Arch())
}

func TestAX7_Client_GetLatestRelease_Bad(t *T) {
	release, err := githubClient{}.GetLatestRelease(Background(), "wrong", "update", "stable")

	AssertError(t, err)
	AssertNil(t, release)
	if err != nil {
		AssertContains(t, err.Error(), "unexpected GitHub request")
	}
}

func TestAX7_Client_GetLatestRelease_Ugly(t *T) {
	release, err := githubClient{}.GetLatestRelease(Background(), "example", "update", " beta ")

	AssertError(t, err)
	AssertNil(t, release)
	if err != nil {
		AssertContains(t, err.Error(), "channel  beta ")
	}
}

func TestAX7_Client_GetReleaseByPullRequest_Good(t *T) {
	release, err := githubClient{}.GetReleaseByPullRequest(Background(), "example", "update", 123)

	AssertNoError(t, err)
	AssertNil(t, release)
}

func TestAX7_Client_GetReleaseByPullRequest_Bad(t *T) {
	release, err := githubClient{}.GetReleaseByPullRequest(Background(), "wrong", "update", -1)

	AssertNoError(t, err)
	AssertNil(t, release)
}

func TestAX7_Client_GetReleaseByPullRequest_Ugly(t *T) {
	ctx, cancel := WithCancel(Background())
	cancel()

	release, err := githubClient{}.GetReleaseByPullRequest(ctx, "", "", 0)

	AssertNoError(t, err)
	AssertNil(t, release)
}
