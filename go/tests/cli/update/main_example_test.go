package main

import (
	. "dappco.re/go"
	updater "dappco.re/go/update"
)

func ExampleClient_GetPublicRepos() {
	result := githubClient{}.GetPublicRepos(Background(), "example")
	Println(result.Value.([]string)[0])
}

func ExampleClient_GetLatestRelease() {
	result := githubClient{}.GetLatestRelease(Background(), "example", "update", "stable")
	Println(result.Value.(*updater.Release).TagName)
}

func ExampleClient_GetReleaseByPullRequest() {
	result := githubClient{}.GetReleaseByPullRequest(Background(), "example", "update", 123)
	Println(result.OK)
}
