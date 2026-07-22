package updater

import (
	"net/http"

	. "dappco.re/go"
)

func TestUnits_StartGitHubCheck_Good(t *T) {
	// CheckAndUpdateOnStartup routes through CheckForUpdates.
	original := CheckForUpdates
	defer func() { CheckForUpdates = original }()
	calls := 0
	CheckForUpdates = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) Result {
		calls++
		return Ok(nil)
	}
	service := &UpdateService{
		config:   UpdateServiceConfig{CheckOnStartup: CheckAndUpdateOnStartup},
		isGitHub: true,
		owner:    "core",
		repo:     "update",
	}

	result := service.Start()

	AssertTrue(t, result.OK)
	AssertEqual(t, 1, calls)
}

func TestUnits_StartGitHubCheck_Bad(t *T) {
	// NoCheck short-circuits to OK with no network work.
	service := &UpdateService{
		config:   UpdateServiceConfig{CheckOnStartup: NoCheck},
		isGitHub: true,
	}

	result := service.Start()

	AssertTrue(t, result.OK)
}

func TestUnits_StartGitHubCheck_Ugly(t *T) {
	// An unknown startup mode fails loudly.
	service := &UpdateService{
		config:   UpdateServiceConfig{CheckOnStartup: StartupCheckMode(42)},
		isGitHub: true,
	}

	result := service.Start()

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown startup check mode")
}

func TestUnits_StartHTTPCheck_Good(t *T) {
	// CheckOnStartup routes through CheckOnlyHTTP.
	original := CheckOnlyHTTP
	defer func() { CheckOnlyHTTP = original }()
	calls := 0
	CheckOnlyHTTP = func(baseURL string) Result {
		calls++
		AssertEqual(t, "https://updates.example.com", baseURL)
		return Ok(nil)
	}
	service := &UpdateService{
		config: UpdateServiceConfig{
			RepoURL:        "https://updates.example.com",
			CheckOnStartup: CheckOnStartup,
		},
		isGitHub: false,
	}

	result := service.Start()

	AssertTrue(t, result.OK)
	AssertEqual(t, 1, calls)
}

func TestUnits_StartHTTPCheck_Bad(t *T) {
	// NoCheck short-circuits to OK.
	service := &UpdateService{
		config:   UpdateServiceConfig{CheckOnStartup: NoCheck},
		isGitHub: false,
	}

	result := service.Start()

	AssertTrue(t, result.OK)
}

func TestUnits_StartHTTPCheck_Ugly(t *T) {
	// An unknown startup mode on the HTTP path also fails loudly.
	service := &UpdateService{
		config:   UpdateServiceConfig{CheckOnStartup: StartupCheckMode(7)},
		isGitHub: false,
	}

	result := service.Start()

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown startup check mode")
}

func TestUnits_DetermineChannel_Good(t *T) {
	AssertEqual(t, "stable", determineChannel("v1.2.3", false))
}

func TestUnits_DetermineChannel_Bad(t *T) {
	// alpha and beta tags map to their channels regardless of the flag.
	AssertEqual(t, "alpha", determineChannel("v1.2.3-alpha.1", false))
	AssertEqual(t, "beta", determineChannel("v1.2.3-beta.2", false))
}

func TestUnits_DetermineChannel_Ugly(t *T) {
	// A prerelease flag without an alpha/beta tag is treated as beta.
	AssertEqual(t, "beta", determineChannel("v1.2.3-rc.1", true))
}

func TestUnits_FindNextURL_Good(t *T) {
	header := `<https://api.github.com/repositories?page=2>; rel="next"`

	AssertEqual(t, "https://api.github.com/repositories?page=2", (&githubClient{}).findNextURL(header))
}

func TestUnits_FindNextURL_Bad(t *T) {
	// No next link present.
	header := `<https://api.github.com/repositories?page=1>; rel="prev"`

	AssertEqual(t, "", (&githubClient{}).findNextURL(header))
}

func TestUnits_FindNextURL_Ugly(t *T) {
	// Empty header yields no URL.
	AssertEqual(t, "", (&githubClient{}).findNextURL(""))
}

func TestUnits_UpdaterUserAgent_Good(t *T) {
	original := Version
	defer func() { Version = original }()
	Version = "1.2.3"

	AssertEqual(t, "agent-go-update/v1.2.3", updaterUserAgent())
}

func TestUnits_UpdaterUserAgent_Bad(t *T) {
	// An empty version yields "agent-go-update/v": formatVersionForDisplay("",
	// true) returns "v" (it adds the prefix to the empty string), so the
	// version=="" "unknown" fallback inside updaterUserAgent is unreachable
	// dead code. Version is initialised from PkgVersion in production, so the
	// empty state never occurs.
	original := Version
	defer func() { Version = original }()
	Version = ""

	AssertEqual(t, "agent-go-update/v", updaterUserAgent())
}

func TestUnits_UpdaterUserAgent_Ugly(t *T) {
	// An already-prefixed version is not double-prefixed.
	original := Version
	defer func() { Version = original }()
	Version = "v9.9.9"

	AssertEqual(t, "agent-go-update/v9.9.9", updaterUserAgent())
}

func TestUnits_NewAgentRequest_Good(t *T) {
	result := newAgentRequest(Background(), "GET", "https://updates.example.com")

	AssertTrue(t, result.OK)
	req := result.Value.(*http.Request)
	AssertContains(t, req.Header.Get("User-Agent"), "agent-go-update/")
}

func TestUnits_NewAgentRequest_Bad(t *T) {
	// A malformed method makes request construction fail.
	result := newAgentRequest(Background(), "BAD METHOD", "https://updates.example.com")

	AssertFalse(t, result.OK)
}

func TestUnits_NewAgentRequest_Ugly(t *T) {
	// A malformed URL also fails before any header is set.
	result := newAgentRequest(Background(), "GET", "://bad-url")

	AssertFalse(t, result.OK)
}

func TestUnits_IsProcessRunning_Good(t *T) {
	// The test process itself is, by definition, running.
	AssertTrue(t, isProcessRunning(Getpid()))
}

func TestUnits_IsProcessRunning_Bad(t *T) {
	// A very high, almost certainly unused PID is not running. A negative PID
	// is deliberately avoided: on Unix syscall.Kill targets a process group
	// and would not exercise the not-running branch.
	AssertFalse(t, isProcessRunning(1<<30))
}

func TestUnits_IsProcessRunning_Ugly(t *T) {
	// Another out-of-range PID at the top of the typical pid_max space.
	AssertFalse(t, isProcessRunning((1<<30)+1))
}
