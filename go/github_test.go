package updater

import (
	"net/http"
	"runtime"

	. "dappco.re/go"
)

type githubTestReadCloser struct {
	Reader
}

func (r githubTestReadCloser) Close() error {
	return nil
}

type githubTestRoundTrip func(*Request) (*Response, error)

func (f githubTestRoundTrip) RoundTrip(req *Request) (*Response, error) {
	return f(req)
}

func githubTestResponse(req *Request, statusCode int, body string) *Response {
	return &Response{
		StatusCode: statusCode,
		Status:     Sprintf("%d %s", statusCode, HTTPStatusText(statusCode)),
		Header:     Header{"Content-Type": []string{"application/json"}},
		Body:       githubTestReadCloser{Reader: NewReader(body)},
		Request:    req,
	}
}

func TestGithub_Client_GetPublicRepos_Good(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			AssertEqual(t, "/users/codex/repos", req.URL.Path)
			return githubTestResponse(req, http.StatusOK, `[{"clone_url":"https://github.com/codex/update.git"}]`), nil
		})}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "codex")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://github.com/codex/update.git"}, result.Value.([]string))
}

func TestGithub_Client_GetPublicRepos_Bad(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusNotFound, ""), nil
		})}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "missing")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to fetch repos")
}

func TestGithub_Client_GetPublicRepos_Ugly(t *T) {
	ctx, cancel := WithCancel(Background())
	cancel()

	result := (&githubClient{}).GetPublicRepos(ctx, "codex")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "canceled")
}

func TestGithub_Client_GetLatestRelease_Good(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			AssertEqual(t, "/repos/core/update/releases", req.URL.Path)
			return githubTestResponse(req, http.StatusOK, `[{"tag_name":"v1.1.0","prerelease":false},{"tag_name":"v1.2.0-beta.1","prerelease":true}]`), nil
		})}
	}

	result := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertTrue(t, result.OK)
	release := result.Value.(*Release)
	AssertEqual(t, "v1.1.0", release.TagName)
}

func TestGithub_Client_GetLatestRelease_Bad(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusInternalServerError, ""), nil
		})}
	}

	result := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to fetch releases")
}

func TestGithub_Client_GetLatestRelease_Ugly(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusOK, `{"tag_name":`), nil
		})}
	}

	result := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertFalse(t, result.OK)
	AssertNotEqual(t, "", result.Error())
}

func TestGithub_Client_GetReleaseByPullRequest_Good(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			AssertEqual(t, "/repos/core/update/releases", req.URL.Path)
			return githubTestResponse(req, http.StatusOK, `[{"tag_name":"v1.2.0-alpha.pr.123","prerelease":true}]`), nil
		})}
	}

	result := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 123)

	AssertTrue(t, result.OK)
	AssertEqual(t, "v1.2.0-alpha.pr.123", result.Value.(*Release).TagName)
}

func TestGithub_Client_GetReleaseByPullRequest_Bad(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusOK, `[{"tag_name":"v1.2.0-alpha.pr.456","prerelease":true}]`), nil
		})}
	}

	result := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 123)

	AssertTrue(t, result.OK)
	AssertNil(t, result.Value.(*Release))
}

func TestGithub_Client_GetReleaseByPullRequest_Ugly(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusOK, `[`), nil
		})}
	}

	result := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 123)

	AssertFalse(t, result.OK)
	AssertNotEqual(t, "", result.Error())
}

func TestGithub_GetDownloadURL_Good(t *T) {
	release := &Release{TagName: "v1.2.3"}

	result := GetDownloadURL(release, "https://updates.example.com/{tag}/{os}/{arch}")

	AssertTrue(t, result.OK)
	AssertContains(t, result.Value.(string), "https://updates.example.com/v1.2.3/")
}

func TestGithub_GetDownloadURL_Bad(t *T) {
	result := GetDownloadURL(nil, "")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "no release provided")
}

func TestGithub_GetDownloadURL_Ugly(t *T) {
	release := &Release{
		TagName: "v1.2.3",
		Assets: []ReleaseAsset{
			{Name: Concat("agent-", runtime.GOOS), DownloadURL: "https://updates.example.com/os-only"},
		},
	}

	result := GetDownloadURL(release, "")

	AssertTrue(t, result.OK)
	AssertEqual(t, "https://updates.example.com/os-only", result.Value.(string))
}
