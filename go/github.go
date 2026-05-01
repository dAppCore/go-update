package updater

import (
	"context"
	"net/http"
	"runtime"

	core "dappco.re/go"
	"golang.org/x/oauth2"
)

// Repo represents a repository from the GitHub API.
type Repo struct {
	CloneURL string `json:"clone_url"` // The URL to clone the repository.
}

// ReleaseAsset represents a single asset from a GitHub release.
type ReleaseAsset struct {
	Name        string `json:"name"`                 // The name of the asset.
	DownloadURL string `json:"browser_download_url"` // The URL to download the asset.
}

// Release represents a GitHub release.
type Release struct {
	TagName    string         `json:"tag_name"`   // The name of the tag for the release.
	PreRelease bool           `json:"prerelease"` // Indicates if the release is a pre-release.
	Assets     []ReleaseAsset `json:"assets"`     // A list of assets associated with the release.
}

// GithubClient defines the interface for interacting with the GitHub API.
// This allows for mocking the client in tests.
type GithubClient interface {
	// GetPublicRepos fetches the public repositories for a user or organization.
	GetPublicRepos(ctx context.Context, userOrOrg string) core.Result
	// GetLatestRelease fetches the latest release for a given repository and channel.
	GetLatestRelease(ctx context.Context, owner, repo, channel string) core.Result
	// GetReleaseByPullRequest fetches a release associated with a specific pull request number.
	GetReleaseByPullRequest(ctx context.Context, owner, repo string, prNumber int) core.Result
}

type githubClient struct{}

// Client exposes the GitHub client method set for examples while the concrete
// implementation remains package-local.
type Client = githubClient

// NewAuthenticatedClient creates a new HTTP client that authenticates with the GitHub API.
// It uses the GITHUB_TOKEN environment variable for authentication.
// If the token is not set, it returns the default HTTP client.
var NewAuthenticatedClient = func(ctx context.Context) *http.Client {
	token := core.Getenv("GITHUB_TOKEN")
	if token == "" {
		return http.DefaultClient
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	client := oauth2.NewClient(ctx, ts)
	client.Timeout = defaultHTTPTimeout
	return client
}

func (g *githubClient) GetPublicRepos(ctx context.Context, userOrOrg string) core.Result {
	return g.getPublicReposWithAPIURL(ctx, "https://api.github.com", userOrOrg)
}

func (g *githubClient) getPublicReposWithAPIURL(ctx context.Context, apiURL, userOrOrg string) core.Result {
	client := NewAuthenticatedClient(ctx)
	var allCloneURLs []string
	url := core.Sprintf("%s/users/%s/repos", apiURL, userOrOrg)

	for {
		if err := ctx.Err(); err != nil {
			return core.Fail(err)
		}

		page := getReposPage(ctx, client, apiURL, userOrOrg, url)
		if !page.OK {
			return page
		}
		resp := page.Value.(*http.Response)

		if resp.StatusCode != http.StatusOK {
			closeResponseBody(resp.Body)
			return core.Fail(core.E("github.getPublicReposWithAPIURL", core.Sprintf("failed to fetch repos: %s", resp.Status), nil))
		}

		var repos []Repo
		body := core.ReadAll(resp.Body)
		if !body.OK {
			return body
		}
		if result := core.JSONUnmarshal([]byte(body.Value.(string)), &repos); !result.OK {
			return result
		}

		for _, repo := range repos {
			allCloneURLs = append(allCloneURLs, repo.CloneURL)
		}

		nextURL := g.findNextURL(resp.Header.Get("Link"))
		if nextURL == "" {
			break
		}
		url = nextURL
	}

	return core.Ok(allCloneURLs)
}

func getReposPage(ctx context.Context, client *http.Client, apiURL, userOrOrg, url string) core.Result {
	first := getURL(ctx, client, url)
	if !first.OK {
		return first
	}
	resp := first.Value.(*http.Response)

	if resp.StatusCode == http.StatusOK {
		return core.Ok(resp)
	}

	closeResponseBody(resp.Body)
	return getURL(ctx, client, core.Sprintf("%s/orgs/%s/repos", apiURL, userOrOrg))
}

func getURL(ctx context.Context, client *http.Client, url string) core.Result {
	request := newAgentRequest(ctx, "GET", url)
	if !request.OK {
		return request
	}
	resp, err := client.Do(request.Value.(*http.Request))
	return core.ResultOf(resp, err)
}

func (g *githubClient) findNextURL(linkHeader string) string {
	links := core.Split(linkHeader, ",")
	for _, link := range links {
		parts := core.Split(link, ";")
		if len(parts) == 2 && core.Trim(parts[1]) == `rel="next"` {
			return core.TrimSuffix(core.TrimPrefix(core.Trim(parts[0]), "<"), ">")
		}
	}
	return ""
}

// GetLatestRelease fetches the latest release for a given repository and channel.
// The channel can be "stable", "beta", or "alpha".
func (g *githubClient) GetLatestRelease(ctx context.Context, owner, repo, channel string) core.Result {
	client := NewAuthenticatedClient(ctx)
	url := core.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	request := newAgentRequest(ctx, "GET", url)
	if !request.OK {
		return request
	}

	resp, err := client.Do(request.Value.(*http.Request))
	if err != nil {
		return core.Fail(err)
	}
	defer closeResponseBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return core.Fail(core.E("github.GetLatestRelease", core.Sprintf("failed to fetch releases: %s", resp.Status), nil))
	}

	var releases []Release
	body := core.ReadAll(resp.Body)
	if !body.OK {
		return body
	}
	if result := core.JSONUnmarshal([]byte(body.Value.(string)), &releases); !result.OK {
		return result
	}

	return core.Ok(filterReleases(releases, channel))
}

// filterReleases filters releases based on the specified channel.
func filterReleases(releases []Release, channel string) *Release {
	for _, release := range releases {
		releaseChannel := determineChannel(release.TagName, release.PreRelease)
		if releaseChannel == channel {
			return &release
		}
	}
	return nil
}

// determineChannel determines the stability channel of a release based on its tag and PreRelease flag.
func determineChannel(tagName string, isPreRelease bool) string {
	tagLower := core.Lower(tagName)
	if core.Contains(tagLower, "alpha") {
		return "alpha"
	}
	if core.Contains(tagLower, "beta") {
		return "beta"
	}
	if isPreRelease { // A pre-release without alpha/beta is treated as beta
		return "beta"
	}
	return "stable"
}

// GetReleaseByPullRequest fetches a release associated with a specific pull request number.
func (g *githubClient) GetReleaseByPullRequest(ctx context.Context, owner, repo string, prNumber int) core.Result {
	client := NewAuthenticatedClient(ctx)
	url := core.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	request := newAgentRequest(ctx, "GET", url)
	if !request.OK {
		return request
	}

	resp, err := client.Do(request.Value.(*http.Request))
	if err != nil {
		return core.Fail(err)
	}
	defer closeResponseBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return core.Fail(core.E("github.GetReleaseByPullRequest", core.Sprintf("failed to fetch releases: %s", resp.Status), nil))
	}

	var releases []Release
	body := core.ReadAll(resp.Body)
	if !body.OK {
		return body
	}
	if result := core.JSONUnmarshal([]byte(body.Value.(string)), &releases); !result.OK {
		return result
	}

	// The pr number is included in the tag name with the format `vX.Y.Z-alpha.pr.123` or `vX.Y.Z-beta.pr.123`
	prTagSuffix := core.Sprintf(".pr.%d", prNumber)
	for _, release := range releases {
		if core.Contains(release.TagName, prTagSuffix) {
			return core.Ok(&release)
		}
	}

	return core.Ok((*Release)(nil)) // No release found for the given PR number
}

// GetDownloadURL finds the appropriate download URL for the current operating system and architecture.
//
// It supports two modes of operation:
//  1. Using a 'releaseURLFormat' template: If 'releaseURLFormat' is provided,
//     it will be used to construct the download URL. The template can contain
//     placeholders for the release tag '{tag}', operating system '{os}', and
//     architecture '{arch}'.
//  2. Automatic detection: If 'releaseURLFormat' is empty, the function will
//     inspect the assets of the release to find a suitable download URL. It
//     searches for an asset name that contains both the current OS and architecture
//     (e.g., "my-app-linux-amd64"). If no match is found, it falls back to
//     matching only the OS.
//
// Example with releaseURLFormat:
//
//	release := &updater.Release{TagName: "v1.2.3"}
//	url, err := updater.GetDownloadURL(release, "https://example.com/downloads/{tag}/{os}/{arch}")
//	if err != nil {
//		// handle error
//	}
//	fmt.Println(url) // "https://example.com/downloads/v1.2.3/linux/amd64" (on a Linux AMD64 system)
//
// Example with automatic detection:
//
//	release := &updater.Release{
//		Assets: []updater.ReleaseAsset{
//			{Name: "my-app-linux-amd64", DownloadURL: "https://example.com/download/linux-amd64"},
//			{Name: "my-app-windows-amd64", DownloadURL: "https://example.com/download/windows-amd64"},
//		},
//	}
//	url, err := updater.GetDownloadURL(release, "")
//	if err != nil {
//		// handle error
//	}
//	fmt.Println(url) // "https://example.com/download/linux-amd64" (on a Linux AMD64 system)
func GetDownloadURL(release *Release, releaseURLFormat string) core.Result {
	if release == nil {
		return core.Fail(core.E("GetDownloadURL", "no release provided", nil))
	}

	if releaseURLFormat != "" {
		url := core.Replace(releaseURLFormat, "{tag}", release.TagName)
		url = core.Replace(url, "{os}", runtime.GOOS)
		url = core.Replace(url, "{arch}", runtime.GOARCH)
		return core.Ok(url)
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH

	for _, asset := range release.Assets {
		assetNameLower := core.Lower(asset.Name)
		// Match asset that contains both OS and architecture
		if core.Contains(assetNameLower, osName) && core.Contains(assetNameLower, archName) {
			return core.Ok(asset.DownloadURL)
		}
	}

	// Fallback for OS only if no asset matched both OS and arch
	for _, asset := range release.Assets {
		assetNameLower := core.Lower(asset.Name)
		if core.Contains(assetNameLower, osName) {
			return core.Ok(asset.DownloadURL)
		}
	}

	return core.Fail(core.E("GetDownloadURL", core.Sprintf("no suitable download asset found for %s/%s", osName, archName), nil))
}
