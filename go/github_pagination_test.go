package updater

import (
	"net/http"

	. "dappco.re/go"
)

// githubErrRoundTrip always fails, exercising transport-error branches.
type githubErrRoundTrip struct{}

func (githubErrRoundTrip) RoundTrip(*Request) (*Response, error) {
	return nil, E("githubErrRoundTrip", "transport boom", nil)
}

func TestGithubPagination_GetPublicRepos_Multipage(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	page := 0
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			page++
			if page == 1 {
				resp := githubTestResponse(req, http.StatusOK, `[{"clone_url":"https://github.com/codex/a.git"}]`)
				resp.Header.Set("Link", `<https://api.github.com/users/codex/repos?page=2>; rel="next"`)
				return resp, nil
			}
			return githubTestResponse(req, http.StatusOK, `[{"clone_url":"https://github.com/codex/b.git"}]`), nil
		})}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "codex")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://github.com/codex/a.git", "https://github.com/codex/b.git"}, result.Value.([]string))
}

func TestGithubPagination_GetPublicRepos_OrgFallback(t *T) {
	// The /users path 404s, so the client retries via /orgs and succeeds.
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			if Contains(req.URL.Path, "/users/") {
				return githubTestResponse(req, http.StatusNotFound, ""), nil
			}
			AssertContains(t, req.URL.Path, "/orgs/")
			return githubTestResponse(req, http.StatusOK, `[{"clone_url":"https://github.com/org/c.git"}]`), nil
		})}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "someorg")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"https://github.com/org/c.git"}, result.Value.([]string))
}

func TestGithubPagination_GetPublicRepos_TransportError(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubErrRoundTrip{}}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "codex")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "transport boom")
}

func TestGithubPagination_GetPublicRepos_BadJSON(t *T) {
	// A 200 with malformed JSON surfaces the unmarshal failure.
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusOK, `[{`), nil
		})}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "codex")

	AssertFalse(t, result.OK)
}

func TestGithubPagination_GetLatestRelease_NoChannelMatch(t *T) {
	// All releases are beta; a stable request returns a nil release, OK.
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusOK, `[{"tag_name":"v1.2.0-beta.1","prerelease":true}]`), nil
		})}
	}

	result := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertTrue(t, result.OK)
	AssertNil(t, result.Value.(*Release))
}

func TestGithubPagination_GetLatestRelease_TransportError(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubErrRoundTrip{}}
	}

	result := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "transport boom")
}

func TestGithubPagination_GetReleaseByPullRequest_BadStatus(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return githubTestResponse(req, http.StatusForbidden, ""), nil
		})}
	}

	result := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 7)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to fetch releases")
}

func TestGithubPagination_GetReleaseByPullRequest_TransportError(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubErrRoundTrip{}}
	}

	result := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 7)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "transport boom")
}

func TestGithubPagination_NewAuthenticatedClient_Token(t *T) {
	// With a GITHUB_TOKEN set the returned client carries the update timeout.
	Setenv("GITHUB_TOKEN", "test-token")
	defer Unsetenv("GITHUB_TOKEN")

	client := NewAuthenticatedClient(Background())

	AssertNotNil(t, client)
	AssertEqual(t, defaultHTTPTimeout, client.Timeout)
}

func TestGithubPagination_NewAuthenticatedClient_NoToken(t *T) {
	// Without a token the default client is returned.
	Unsetenv("GITHUB_TOKEN")

	client := NewAuthenticatedClient(Background())

	AssertNotNil(t, client)
}
