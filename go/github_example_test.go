package updater

import . "dappco.re/go"

func ExampleClient_GetPublicRepos() {
	result := (&githubClient{}).findNextURL(`<https://api.github.com/repositories?page=2>; rel="next"`)
	Println(result)
}

func ExampleClient_GetLatestRelease() {
	release := filterReleases([]Release{{TagName: "v1.0.0"}}, "stable")
	Println(release.TagName)
}

func ExampleClient_GetReleaseByPullRequest() {
	tag := Sprintf("v1.2.0-alpha.pr.%d", 123)
	Println(Contains(tag, ".pr.123"))
}

func ExampleGetDownloadURL() {
	result := GetDownloadURL(&Release{TagName: "v1.2.3"}, "https://updates.example.com/{tag}")
	Println(result.Value.(string))
}
