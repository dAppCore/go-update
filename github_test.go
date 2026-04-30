package updater

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	// Note: Borg encryption library; awaiting canonical dappco.re/go/borg publication.
	"github.com/Snider/Borg/pkg/mocks"
)

const (
	githubTestAPIURL            = "https://api.github.com"
	githubTestUserReposURL      = githubTestAPIURL + "/users/testuser/repos"
	githubTestContentTypeHeader = "Content-Type"
	githubTestApplicationJSON   = "application/json"
)

func TestGetPublicRepos(t *testing.T) {
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		githubTestUserReposURL: {
			StatusCode: http.StatusOK,
			Header:     http.Header{githubTestContentTypeHeader: []string{githubTestApplicationJSON}},
			Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testuser/repo1.git"}]`)),
		},
		githubTestAPIURL + "/orgs/testorg/repos": {
			StatusCode: http.StatusOK,
			Header:     http.Header{githubTestContentTypeHeader: []string{githubTestApplicationJSON}, "Link": []string{`<https://api.github.com/organizations/123/repos?page=2>; rel="next"`}},
			Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testorg/repo1.git"}]`)),
		},
		githubTestAPIURL + "/organizations/123/repos?page=2": {
			StatusCode: http.StatusOK,
			Header:     http.Header{githubTestContentTypeHeader: []string{githubTestApplicationJSON}},
			Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testorg/repo2.git"}]`)),
		},
	})

	client := &githubClient{}
	oldClient := NewAuthenticatedClient
	NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return mockClient
	}
	defer func() {
		NewAuthenticatedClient = oldClient
	}()

	// Test user repos
	repos, err := client.getPublicReposWithAPIURL(context.Background(), githubTestAPIURL, "testuser")
	if err != nil {
		t.Fatalf("getPublicReposWithAPIURL for user failed: %v", err)
	}
	if len(repos) != 1 || repos[0] != "https://github.com/testuser/repo1.git" {
		t.Errorf("unexpected user repos: %v", repos)
	}

	// Test org repos with pagination
	repos, err = client.getPublicReposWithAPIURL(context.Background(), githubTestAPIURL, "testorg")
	if err != nil {
		t.Fatalf("getPublicReposWithAPIURL for org failed: %v", err)
	}
	if len(repos) != 2 || repos[0] != "https://github.com/testorg/repo1.git" || repos[1] != "https://github.com/testorg/repo2.git" {
		t.Errorf("unexpected org repos: %v", repos)
	}
}
func TestGetPublicRepos_Error(t *testing.T) {
	u, _ := url.Parse(githubTestUserReposURL)
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		githubTestUserReposURL: {
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Header:     http.Header{githubTestContentTypeHeader: []string{githubTestApplicationJSON}},
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Request:    &http.Request{Method: "GET", URL: u},
		},
		githubTestAPIURL + "/orgs/testuser/repos": {
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Header:     http.Header{githubTestContentTypeHeader: []string{githubTestApplicationJSON}},
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Request:    &http.Request{Method: "GET", URL: u},
		},
	})
	expectedErr := "github.getPublicReposWithAPIURL: failed to fetch repos: 404 Not Found"

	client := &githubClient{}
	oldClient := NewAuthenticatedClient
	NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return mockClient
	}
	defer func() {
		NewAuthenticatedClient = oldClient
	}()

	// Test user repos
	_, err := client.getPublicReposWithAPIURL(context.Background(), githubTestAPIURL, "testuser")
	if err.Error() != expectedErr {
		t.Fatalf("getPublicReposWithAPIURL for user failed: expected %q, got %q", expectedErr, err.Error())
	}
}

func TestFindNextURL(t *testing.T) {
	client := &githubClient{}
	linkHeader := `<https://api.github.com/organizations/123/repos?page=2>; rel="next", <https://api.github.com/organizations/123/repos?page=1>; rel="prev"`
	nextURL := client.findNextURL(linkHeader)
	if nextURL != "https://api.github.com/organizations/123/repos?page=2" {
		t.Errorf("unexpected next URL: %s", nextURL)
	}

	linkHeader = `<https://api.github.com/organizations/123/repos?page=1>; rel="prev"`
	nextURL = client.findNextURL(linkHeader)
	if nextURL != "" {
		t.Errorf("unexpected next URL: %s", nextURL)
	}
}

func TestNewAuthenticatedClient(t *testing.T) {
	// Test with no token
	client := NewAuthenticatedClient(context.Background())
	if client != http.DefaultClient {
		t.Errorf("expected http.DefaultClient, but got something else")
	}

	// Test with token
	t.Setenv("GITHUB_TOKEN", "test-token")
	client = NewAuthenticatedClient(context.Background())
	if client == http.DefaultClient {
		t.Errorf("expected an authenticated client, but got http.DefaultClient")
	}
}
