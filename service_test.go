package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const serviceTestGitHubRepoURL = "https://github.com/owner/repo"

type updateServiceStartCase struct {
	name                string
	config              UpdateServiceConfig
	checkOnlyGitHub     int
	checkAndDoGitHub    int
	checkOnlyHTTPCalls  int
	checkAndDoHTTPCalls int
	expectError         bool
}

type updateServiceStartCalls struct {
	checkOnlyGitHub  int
	checkAndDoGitHub int
	checkOnlyHTTP    int
	checkAndDoHTTP   int
}

func TestNewUpdateService(t *testing.T) {
	testCases := []struct {
		name        string
		config      UpdateServiceConfig
		expectError bool
		isGitHub    bool
		wantChannel string
	}{
		{
			name: "Valid GitHub URL",
			config: UpdateServiceConfig{
				RepoURL: serviceTestGitHubRepoURL,
			},
			isGitHub:    true,
			wantChannel: "stable",
		},
		{
			name: "Valid non-GitHub URL",
			config: UpdateServiceConfig{
				RepoURL: "https://example.com/updates",
			},
			isGitHub: false,
		},
		{
			name: "GitHub channel is normalised",
			config: UpdateServiceConfig{
				RepoURL: serviceTestGitHubRepoURL,
				Channel: " Beta ",
			},
			isGitHub:    true,
			wantChannel: "beta",
		},
		{
			name: "GitHub prerelease channel maps to beta",
			config: UpdateServiceConfig{
				RepoURL: serviceTestGitHubRepoURL,
				Channel: " prerelease ",
			},
			isGitHub:    true,
			wantChannel: "beta",
		},
		{
			name: "Invalid GitHub URL",
			config: UpdateServiceConfig{
				RepoURL: "https://github.com/owner",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewUpdateService(tc.config)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if err == nil && service.isGitHub != tc.isGitHub {
				t.Errorf("Expected isGitHub: %v, got: %v", tc.isGitHub, service.isGitHub)
			}
			if err == nil && tc.wantChannel != "" && service.config.Channel != tc.wantChannel {
				t.Errorf("Expected GitHub channel %q, got %q", tc.wantChannel, service.config.Channel)
			}
		})
	}
}

func TestUpdateService_Start(t *testing.T) {
	// Setup a mock server for HTTP tests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version": "v1.1.0", "url": "http://example.com/release.zip"}`))
	}))
	defer server.Close()

	testCases := []updateServiceStartCase{
		{
			name: "GitHub: NoCheck",
			config: UpdateServiceConfig{
				RepoURL:        serviceTestGitHubRepoURL,
				CheckOnStartup: NoCheck,
			},
		},
		{
			name: "GitHub: CheckOnStartup",
			config: UpdateServiceConfig{
				RepoURL:        serviceTestGitHubRepoURL,
				CheckOnStartup: CheckOnStartup,
			},
			checkOnlyGitHub: 1,
		},
		{
			name: "GitHub: CheckAndUpdateOnStartup",
			config: UpdateServiceConfig{
				RepoURL:        serviceTestGitHubRepoURL,
				CheckOnStartup: CheckAndUpdateOnStartup,
			},
			checkAndDoGitHub: 1,
		},
		{
			name: "HTTP: NoCheck",
			config: UpdateServiceConfig{
				RepoURL:        server.URL,
				CheckOnStartup: NoCheck,
			},
		},
		{
			name: "HTTP: CheckOnStartup",
			config: UpdateServiceConfig{
				RepoURL:        server.URL,
				CheckOnStartup: CheckOnStartup,
			},
			checkOnlyHTTPCalls: 1,
		},
		{
			name: "HTTP: CheckAndUpdateOnStartup",
			config: UpdateServiceConfig{
				RepoURL:        server.URL,
				CheckOnStartup: CheckAndUpdateOnStartup,
			},
			checkAndDoHTTPCalls: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runUpdateServiceStartCase(t, tc)
		})
	}
}

func runUpdateServiceStartCase(t *testing.T, tc updateServiceStartCase) {
	t.Helper()

	calls := stubUpdateChecks(t)
	service, _ := NewUpdateService(tc.config)
	err := service.Start()

	assertUpdateServiceStartResult(t, tc, calls, err)
}

func stubUpdateChecks(t *testing.T) *updateServiceStartCalls {
	t.Helper()

	calls := &updateServiceStartCalls{}

	originalCheckOnly := CheckOnly
	CheckOnly = func(_, _, _ string, _ bool, _ string) error {
		calls.checkOnlyGitHub++
		return nil
	}
	t.Cleanup(func() { CheckOnly = originalCheckOnly })

	originalCheckForUpdates := CheckForUpdates
	CheckForUpdates = func(_, _, _ string, _ bool, _ string) error {
		calls.checkAndDoGitHub++
		return nil
	}
	t.Cleanup(func() { CheckForUpdates = originalCheckForUpdates })

	originalCheckOnlyHTTP := CheckOnlyHTTP
	CheckOnlyHTTP = func(_ string) error {
		calls.checkOnlyHTTP++
		return nil
	}
	t.Cleanup(func() { CheckOnlyHTTP = originalCheckOnlyHTTP })

	originalCheckForUpdatesHTTP := CheckForUpdatesHTTP
	CheckForUpdatesHTTP = func(_ string) error {
		calls.checkAndDoHTTP++
		return nil
	}
	t.Cleanup(func() { CheckForUpdatesHTTP = originalCheckForUpdatesHTTP })

	return calls
}

func assertUpdateServiceStartResult(t *testing.T, tc updateServiceStartCase, calls *updateServiceStartCalls, err error) {
	t.Helper()

	if (err != nil) != tc.expectError {
		t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
	}
	if calls.checkOnlyGitHub != tc.checkOnlyGitHub {
		t.Errorf("Expected GitHub CheckOnly calls: %d, got: %d", tc.checkOnlyGitHub, calls.checkOnlyGitHub)
	}
	if calls.checkAndDoGitHub != tc.checkAndDoGitHub {
		t.Errorf("Expected GitHub CheckForUpdates calls: %d, got: %d", tc.checkAndDoGitHub, calls.checkAndDoGitHub)
	}
	if calls.checkOnlyHTTP != tc.checkOnlyHTTPCalls {
		t.Errorf("Expected HTTP CheckOnly calls: %d, got: %d", tc.checkOnlyHTTPCalls, calls.checkOnlyHTTP)
	}
	if calls.checkAndDoHTTP != tc.checkAndDoHTTPCalls {
		t.Errorf("Expected HTTP CheckForUpdates calls: %d, got: %d", tc.checkAndDoHTTPCalls, calls.checkAndDoHTTP)
	}
}
