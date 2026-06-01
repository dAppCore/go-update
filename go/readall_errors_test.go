package updater

import (
	"net/http"

	. "dappco.re/go"
)

// errReadCloser fails on Read, exercising the core.ReadAll error branches in
// the GitHub and generic-HTTP response handlers.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, E("errReadCloser", "read boom", nil) }
func (errReadCloser) Close() error             { return nil }

func errBodyResponse(req *Request, status int) *Response {
	return &Response{
		StatusCode: status,
		Status:     Sprintf("%d %s", status, HTTPStatusText(status)),
		Header:     Header{"Content-Type": []string{"application/json"}},
		Body:       errReadCloser{},
		Request:    req,
	}
}

func TestReadAllErrors_GetLatestRelease_BodyReadError(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return errBodyResponse(req, http.StatusOK), nil
		})}
	}

	result := (&githubClient{}).GetLatestRelease(Background(), "core", "update", "stable")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "read boom")
}

func TestReadAllErrors_GetReleaseByPullRequest_BodyReadError(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return errBodyResponse(req, http.StatusOK), nil
		})}
	}

	result := (&githubClient{}).GetReleaseByPullRequest(Background(), "core", "update", 1)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "read boom")
}

func TestReadAllErrors_GetPublicRepos_BodyReadError(t *T) {
	original := NewAuthenticatedClient
	defer func() { NewAuthenticatedClient = original }()
	NewAuthenticatedClient = func(_ Context) *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return errBodyResponse(req, http.StatusOK), nil
		})}
	}

	result := (&githubClient{}).GetPublicRepos(Background(), "codex")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "read boom")
}

func TestReadAllErrors_GetLatestUpdateFromURL_BodyReadError(t *T) {
	original := NewHTTPClient
	defer func() { NewHTTPClient = original }()
	NewHTTPClient = func() *http.Client {
		return &http.Client{Transport: githubTestRoundTrip(func(req *Request) (*Response, error) {
			return errBodyResponse(req, http.StatusOK), nil
		})}
	}

	result := GetLatestUpdateFromURL("https://updates.example.com")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to read latest.json")
}
