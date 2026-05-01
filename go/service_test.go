package updater

import . "dappco.re/go"

const serviceTestRepo = "https://github.com/core/update"

func TestService_NewUpdateService_Good(t *T) {
	result := NewUpdateService(UpdateServiceConfig{RepoURL: serviceTestRepo, Channel: " prerelease "})

	AssertTrue(t, result.OK)
	service := result.Value.(*UpdateService)
	AssertTrue(t, service.isGitHub)
	AssertEqual(t, "core", service.owner)
	AssertEqual(t, "update", service.repo)
	AssertEqual(t, "beta", service.config.Channel)
}

func TestService_NewUpdateService_Bad(t *T) {
	result := NewUpdateService(UpdateServiceConfig{RepoURL: "https://github.com/core"})

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to parse GitHub repo URL")
}

func TestService_NewUpdateService_Ugly(t *T) {
	result := NewUpdateService(UpdateServiceConfig{RepoURL: "https://updates.example.com/releases"})

	AssertTrue(t, result.OK)
	service := result.Value.(*UpdateService)
	AssertFalse(t, service.isGitHub)
	AssertEqual(t, "", service.owner)
	AssertEqual(t, "", service.repo)
}

func TestService_UpdateService_Start_Good(t *T) {
	original := CheckOnly
	defer func() { CheckOnly = original }()
	calls := 0
	CheckOnly = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) Result {
		calls++
		AssertEqual(t, "core", owner)
		AssertEqual(t, "update", repo)
		AssertEqual(t, "stable", channel)
		return Ok(nil)
	}
	service := NewUpdateService(UpdateServiceConfig{RepoURL: serviceTestRepo, CheckOnStartup: CheckOnStartup}).Value.(*UpdateService)

	result := service.Start()

	AssertTrue(t, result.OK)
	AssertEqual(t, 1, calls)
}

func TestService_UpdateService_Start_Bad(t *T) {
	service := &UpdateService{config: UpdateServiceConfig{CheckOnStartup: StartupCheckMode(99)}, isGitHub: true}

	result := service.Start()

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown startup check mode")
}

func TestService_UpdateService_Start_Ugly(t *T) {
	original := CheckForUpdatesHTTP
	defer func() { CheckForUpdatesHTTP = original }()
	calls := 0
	CheckForUpdatesHTTP = func(baseURL string) Result {
		calls++
		AssertEqual(t, "https://updates.example.com", baseURL)
		return Ok(nil)
	}
	service := NewUpdateService(UpdateServiceConfig{RepoURL: "https://updates.example.com", CheckOnStartup: CheckAndUpdateOnStartup}).Value.(*UpdateService)

	result := service.Start()

	AssertTrue(t, result.OK)
	AssertEqual(t, 1, calls)
}

func TestService_ParseRepoURL_Good(t *T) {
	result := ParseRepoURL(serviceTestRepo)

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"core", "update"}, result.Value.([]string))
}

func TestService_ParseRepoURL_Bad(t *T) {
	result := ParseRepoURL("://bad-url")

	AssertFalse(t, result.OK)
	AssertNotEqual(t, "", result.Error())
}

func TestService_ParseRepoURL_Ugly(t *T) {
	result := ParseRepoURL(serviceTestRepo + "/tree/main")

	AssertTrue(t, result.OK)
	AssertEqual(t, []string{"core", "update"}, result.Value.([]string))
}
