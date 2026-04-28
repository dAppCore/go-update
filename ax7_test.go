package updater

import (
	. "dappco.re/go"
	"github.com/spf13/cobra"
)

type ax7ReadCloser struct {
	Reader
}

func (r ax7ReadCloser) Close() error {
	return nil
}

type ax7RoundTrip func(*Request) (*Response, error)

func (f ax7RoundTrip) RoundTrip(req *Request) (*Response, error) {
	return f(req)
}

func ax7HTTPResponse(req *Request, statusCode int, body string) *Response {
	return &Response{
		StatusCode: statusCode,
		Status:     Sprintf("%d %s", statusCode, HTTPStatusText(statusCode)),
		Header:     Header{"Content-Type": []string{"application/json"}},
		Body:       ax7ReadCloser{Reader: NewReader(body)},
		Request:    req,
	}
}

func TestAX7_AddUpdateCommands_Good(t *T) {
	root := &cobra.Command{Use: "core"}
	AddUpdateCommands(root)
	cmd, _, err := root.Find([]string{"update"})

	AssertNoError(t, err)
	AssertEqual(t, "update", cmd.Use)
	AssertNotNil(t, cmd.Flags().Lookup("check"))
}

func TestAX7_AddUpdateCommands_Bad(t *T) {
	var root *cobra.Command

	AssertPanics(t, func() {
		AddUpdateCommands(root)
	})
}

func TestAX7_AddUpdateCommands_Ugly(t *T) {
	root := &cobra.Command{Use: "core"}
	AddUpdateCommands(root)
	cmd, _, err := root.Find([]string{"update"})

	AssertNoError(t, err)
	flag := cmd.Flags().Lookup("watch-pid")
	AssertNotNil(t, flag)
	AssertTrue(t, flag.Hidden)
}

func TestAX7_Client_GetPublicRepos_Good(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			AssertEqual(t, "/users/codex/repos", req.URL.Path)
			return ax7HTTPResponse(req, 200, `[{"clone_url":"https://github.com/codex/update.git"}]`), nil
		})}
	}

	repos, err := (&githubClient{}).GetPublicRepos(Background(), "codex")

	AssertNoError(t, err)
	AssertEqual(t, []string{"https://github.com/codex/update.git"}, repos)
}

func TestAX7_Client_GetPublicRepos_Bad(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			return ax7HTTPResponse(req, 404, ""), nil
		})}
	}

	repos, err := (&githubClient{}).GetPublicRepos(Background(), "missing")

	AssertError(t, err)
	AssertNil(t, repos)
	if err != nil {
		AssertContains(t, err.Error(), "failed to fetch repos")
	}
}

func TestAX7_Client_GetPublicRepos_Ugly(t *T) {
	ctx, cancel := WithCancel(Background())
	cancel()

	repos, err := (&githubClient{}).GetPublicRepos(ctx, "codex")

	AssertError(t, err)
	AssertNil(t, repos)
	if err != nil {
		AssertContains(t, err.Error(), "canceled")
	}
}

func TestAX7_Client_GetLatestRelease_Good(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			AssertEqual(t, "/repos/core/update/releases", req.URL.Path)
			return ax7HTTPResponse(req, 200, `[{"tag_name":"v1.1.0","prerelease":false},{"tag_name":"v1.2.0-beta.1","prerelease":true}]`), nil
		})}
	}

	release, err := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertNoError(t, err)
	AssertNotNil(t, release)
	AssertEqual(t, "v1.1.0", release.TagName)
}

func TestAX7_Client_GetLatestRelease_Bad(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			return ax7HTTPResponse(req, 500, ""), nil
		})}
	}

	release, err := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertError(t, err)
	AssertNil(t, release)
	if err != nil {
		AssertContains(t, err.Error(), "failed to fetch releases")
	}
}

func TestAX7_Client_GetLatestRelease_Ugly(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			return ax7HTTPResponse(req, 200, `{"tag_name":`), nil
		})}
	}

	release, err := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertError(t, err)
	AssertNil(t, release)
}

func TestAX7_Client_GetReleaseByPullRequest_Good(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			AssertEqual(t, "/repos/core/update/releases", req.URL.Path)
			return ax7HTTPResponse(req, 200, `[{"tag_name":"v1.2.0-alpha.pr.123","prerelease":true}]`), nil
		})}
	}

	release, err := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 123)

	AssertNoError(t, err)
	AssertNotNil(t, release)
	AssertEqual(t, "v1.2.0-alpha.pr.123", release.TagName)
}

func TestAX7_Client_GetReleaseByPullRequest_Bad(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			return ax7HTTPResponse(req, 200, `[{"tag_name":"v1.2.0-alpha.pr.456","prerelease":true}]`), nil
		})}
	}

	release, err := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 123)

	AssertNoError(t, err)
	AssertNil(t, release)
}

func TestAX7_Client_GetReleaseByPullRequest_Ugly(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *HTTPClient {
		return &HTTPClient{Transport: ax7RoundTrip(func(req *Request) (*Response, error) {
			return ax7HTTPResponse(req, 200, `[`), nil
		})}
	}

	release, err := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 123)

	AssertError(t, err)
	AssertNil(t, release)
}

func TestAX7_GetDownloadURL_Good(t *T) {
	release := &Release{TagName: "v1.2.3"}
	url, err := GetDownloadURL(release, "https://updates.example.com/{tag}/{os}/{arch}")

	AssertNoError(t, err)
	AssertContains(t, url, "https://updates.example.com/v1.2.3/")
}

func TestAX7_GetDownloadURL_Bad(t *T) {
	url, err := GetDownloadURL(nil, "")

	AssertError(t, err)
	AssertEqual(t, "", url)
	if err != nil {
		AssertContains(t, err.Error(), "no release provided")
	}
}

func TestAX7_GetDownloadURL_Ugly(t *T) {
	release := &Release{
		TagName: "v1.2.3",
		Assets: []ReleaseAsset{
			{Name: Concat("agent-", "other"), DownloadURL: "https://updates.example.com/other"},
			{Name: Concat("agent-", OS()), DownloadURL: "https://updates.example.com/os-only"},
		},
	}

	url, err := GetDownloadURL(release, "")

	AssertNoError(t, err)
	AssertEqual(t, "https://updates.example.com/os-only", url)
}

func TestAX7_GetLatestUpdateFromURL_Good(t *T) {
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		AssertEqual(t, "/latest.json", r.URL.Path)
		write := WriteString(w, `{"version":"v1.2.0","url":"https://updates.example.com/app"}`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	info, err := GetLatestUpdateFromURL(server.URL)

	AssertNoError(t, err)
	AssertEqual(t, "v1.2.0", info.Version)
	AssertEqual(t, "https://updates.example.com/app", info.URL)
}

func TestAX7_GetLatestUpdateFromURL_Bad(t *T) {
	info, err := GetLatestUpdateFromURL("://bad-url")

	AssertError(t, err)
	AssertNil(t, info)
}

func TestAX7_GetLatestUpdateFromURL_Ugly(t *T) {
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		AssertEqual(t, "/latest.json", r.URL.Path)
		write := WriteString(w, `{"version":`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	info, err := GetLatestUpdateFromURL(server.URL)

	AssertError(t, err)
	AssertNil(t, info)
}

func TestAX7_NewUpdateService_Good(t *T) {
	service, err := NewUpdateService(UpdateServiceConfig{
		RepoURL: "https://github.com/core/update",
		Channel: " prerelease ",
	})

	AssertNoError(t, err)
	AssertTrue(t, service.isGitHub)
	AssertEqual(t, "core", service.owner)
	AssertEqual(t, "update", service.repo)
	AssertEqual(t, "beta", service.config.Channel)
}

func TestAX7_NewUpdateService_Bad(t *T) {
	service, err := NewUpdateService(UpdateServiceConfig{RepoURL: "https://github.com/core"})

	AssertError(t, err)
	AssertNil(t, service)
}

func TestAX7_NewUpdateService_Ugly(t *T) {
	service, err := NewUpdateService(UpdateServiceConfig{
		RepoURL:        "https://updates.example.com/releases",
		CheckOnStartup: NoCheck,
	})

	AssertNoError(t, err)
	AssertFalse(t, service.isGitHub)
	AssertEqual(t, "", service.owner)
	AssertEqual(t, "", service.repo)
}

func TestAX7_ParseRepoURL_Good(t *T) {
	owner, repo, err := ParseRepoURL("https://github.com/core/update")

	AssertNoError(t, err)
	AssertEqual(t, "core", owner)
	AssertEqual(t, "update", repo)
}

func TestAX7_ParseRepoURL_Bad(t *T) {
	owner, repo, err := ParseRepoURL("://bad-url")

	AssertError(t, err)
	AssertEqual(t, "", owner)
	AssertEqual(t, "", repo)
}

func TestAX7_ParseRepoURL_Ugly(t *T) {
	owner, repo, err := ParseRepoURL("https://github.com/core/update/tree/main")

	AssertNoError(t, err)
	AssertEqual(t, "core", owner)
	AssertEqual(t, "update", repo)
}

func TestAX7_UpdateService_Start_Good(t *T) {
	original := CheckOnly
	defer func() { CheckOnly = original }()
	calls := 0
	CheckOnly = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) error {
		calls++
		AssertEqual(t, "core", owner)
		AssertEqual(t, "update", repo)
		AssertEqual(t, "stable", channel)
		return nil
	}
	service, err := NewUpdateService(UpdateServiceConfig{
		RepoURL:        "https://github.com/core/update",
		CheckOnStartup: CheckOnStartup,
	})

	AssertNoError(t, err)
	AssertNoError(t, service.Start())
	AssertEqual(t, 1, calls)
}

func TestAX7_UpdateService_Start_Bad(t *T) {
	service := &UpdateService{
		config:   UpdateServiceConfig{CheckOnStartup: StartupCheckMode(99)},
		isGitHub: true,
		owner:    "core",
		repo:     "update",
	}

	err := service.Start()

	AssertError(t, err)
	if err != nil {
		AssertContains(t, err.Error(), "unknown startup check mode")
	}
}

func TestAX7_UpdateService_Start_Ugly(t *T) {
	original := CheckForUpdatesHTTP
	defer func() { CheckForUpdatesHTTP = original }()
	calls := 0
	CheckForUpdatesHTTP = func(baseURL string) error {
		calls++
		AssertEqual(t, "https://updates.example.com", baseURL)
		return nil
	}
	service, err := NewUpdateService(UpdateServiceConfig{
		RepoURL:        "https://updates.example.com",
		CheckOnStartup: CheckAndUpdateOnStartup,
	})

	AssertNoError(t, err)
	AssertNoError(t, service.Start())
	AssertEqual(t, 1, calls)
}
