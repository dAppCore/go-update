package updater

import (
	"runtime"
	"testing"
)

const (
	tagV100Alpha          = "v1.0.0-alpha.1"
	tagV100Beta           = "v1.0.0-beta.1"
	tagV100               = "v1.0.0"
	tagV200RC1            = "v2.0.0-rc.1"
	unexpectedErrorFormat = "unexpected error: %v"
	tagV123               = "v1.2.3"
	osOnlyDownloadURL     = "https://example.com/os-only"
)

func TestFilterReleases_Good(t *testing.T) {
	releases := []Release{
		{TagName: tagV100Alpha, PreRelease: true},
		{TagName: tagV100Beta, PreRelease: true},
		{TagName: tagV100, PreRelease: false},
	}

	tests := []struct {
		channel string
		wantTag string
	}{
		{"stable", tagV100},
		{"alpha", tagV100Alpha},
		{"beta", tagV100Beta},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			got := filterReleases(releases, tt.channel)
			if got == nil {
				t.Fatalf("expected release for channel %q, got nil", tt.channel)
			}
			if got.TagName != tt.wantTag {
				t.Errorf("expected tag %q, got %q", tt.wantTag, got.TagName)
			}
		})
	}
}

func TestFilterReleases_Bad(t *testing.T) {
	releases := []Release{
		{TagName: tagV100, PreRelease: false},
	}
	got := filterReleases(releases, "alpha")
	if got != nil {
		t.Errorf("expected nil for non-matching channel, got %v", got)
	}
}

func TestFilterReleases_PreReleaseWithoutLabel(t *testing.T) {
	releases := []Release{
		{TagName: tagV200RC1, PreRelease: true},
	}
	got := filterReleases(releases, "beta")
	if got == nil {
		t.Fatal("expected pre-release without alpha/beta label to match beta channel")
	}
	if got.TagName != tagV200RC1 {
		t.Errorf("expected tag %q, got %q", tagV200RC1, got.TagName)
	}
}

func TestDetermineChannel_Good(t *testing.T) {
	tests := []struct {
		tag          string
		isPreRelease bool
		want         string
	}{
		{tagV100, false, "stable"},
		{tagV100Alpha, false, "alpha"},
		{"v1.0.0-ALPHA.1", false, "alpha"},
		{tagV100Beta, false, "beta"},
		{"v1.0.0-BETA.1", false, "beta"},
		{"v1.0.0-rc.1", true, "beta"},
		{"v1.0.0-rc.1", false, "stable"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := determineChannel(tt.tag, tt.isPreRelease)
			if got != tt.want {
				t.Errorf("determineChannel(%q, %v) = %q, want %q", tt.tag, tt.isPreRelease, got, tt.want)
			}
		})
	}
}

func TestCheckForUpdatesByTag_UsesCurrentVersionChannel(t *testing.T) {
	originalVersion := Version
	originalCheckForUpdates := CheckForUpdates
	defer func() {
		Version = originalVersion
		CheckForUpdates = originalCheckForUpdates
	}()

	var gotChannel string
	CheckForUpdates = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) error {
		gotChannel = channel
		return nil
	}

	Version = tagV200RC1
	if err := CheckForUpdatesByTag("owner", "repo"); err != nil {
		t.Fatalf(unexpectedErrorFormat, err)
	}

	if gotChannel != "beta" {
		t.Fatalf("expected beta channel, got %q", gotChannel)
	}
}

func TestCheckOnlyByTag_UsesCurrentVersionChannel(t *testing.T) {
	originalVersion := Version
	originalCheckOnly := CheckOnly
	defer func() {
		Version = originalVersion
		CheckOnly = originalCheckOnly
	}()

	var gotChannel string
	CheckOnly = func(owner, repo, channel string, forceSemVerPrefix bool, releaseURLFormat string) error {
		gotChannel = channel
		return nil
	}

	Version = "v2.0.0-alpha.1"
	if err := CheckOnlyByTag("owner", "repo"); err != nil {
		t.Fatalf(unexpectedErrorFormat, err)
	}

	if gotChannel != "alpha" {
		t.Fatalf("expected alpha channel, got %q", gotChannel)
	}
}

func TestGetDownloadURL_Good(t *testing.T) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	release := &Release{
		TagName: tagV123,
		Assets: []ReleaseAsset{
			{Name: "app-" + osName + "-" + archName, DownloadURL: "https://example.com/full-match"},
			{Name: "app-" + osName, DownloadURL: osOnlyDownloadURL},
		},
	}

	url, err := GetDownloadURL(release, "")
	if err != nil {
		t.Fatalf(unexpectedErrorFormat, err)
	}
	if url != "https://example.com/full-match" {
		t.Errorf("expected full match URL, got %q", url)
	}
}

func TestGetDownloadURL_OSOnlyFallback(t *testing.T) {
	osName := runtime.GOOS

	release := &Release{
		TagName: tagV123,
		Assets: []ReleaseAsset{
			{Name: "app-other-other", DownloadURL: "https://example.com/other"},
			{Name: "app-" + osName + "-other", DownloadURL: osOnlyDownloadURL},
		},
	}

	url, err := GetDownloadURL(release, "")
	if err != nil {
		t.Fatalf(unexpectedErrorFormat, err)
	}
	if url != osOnlyDownloadURL {
		t.Errorf("expected OS-only fallback URL, got %q", url)
	}
}

func TestGetDownloadURL_WithFormat(t *testing.T) {
	release := &Release{TagName: tagV123}

	url, err := GetDownloadURL(release, "https://example.com/{tag}/{os}/{arch}")
	if err != nil {
		t.Fatalf(unexpectedErrorFormat, err)
	}

	expected := "https://example.com/" + tagV123 + "/" + runtime.GOOS + "/" + runtime.GOARCH
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestGetDownloadURL_Bad(t *testing.T) {
	// nil release
	_, err := GetDownloadURL(nil, "")
	if err == nil {
		t.Error("expected error for nil release")
	}

	// No matching assets
	release := &Release{
		TagName: tagV123,
		Assets: []ReleaseAsset{
			{Name: "app-unknownos-unknownarch", DownloadURL: "https://example.com/other"},
		},
	}
	_, err = GetDownloadURL(release, "")
	if err == nil {
		t.Error("expected error when no suitable asset found")
	}
}

func TestFormatVersionForComparison(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.0", tagV100},
		{tagV100, tagV100},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatVersionForComparison(tt.input)
			if got != tt.want {
				t.Errorf("formatVersionForComparison(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatVersionForDisplay(t *testing.T) {
	tests := []struct {
		version string
		force   bool
		want    string
	}{
		{"1.0.0", true, tagV100},
		{tagV100, true, tagV100},
		{tagV100, false, "1.0.0"},
		{"1.0.0", false, "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_force_"+boolStr(tt.force), func(t *testing.T) {
			got := formatVersionForDisplay(tt.version, tt.force)
			if got != tt.want {
				t.Errorf("formatVersionForDisplay(%q, %v) = %q, want %q", tt.version, tt.force, got, tt.want)
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestStartGitHubCheck_UnknownMode(t *testing.T) {
	s := &UpdateService{
		config: UpdateServiceConfig{
			CheckOnStartup: StartupCheckMode(99),
		},
		isGitHub: true,
		owner:    "owner",
		repo:     "repo",
	}
	err := s.Start()
	if err == nil {
		t.Error("expected error for unknown startup check mode")
	}
}

func TestStartHTTPCheck_UnknownMode(t *testing.T) {
	s := &UpdateService{
		config: UpdateServiceConfig{
			RepoURL:        "https://example.com/updates",
			CheckOnStartup: StartupCheckMode(99),
		},
		isGitHub: false,
	}
	err := s.Start()
	if err == nil {
		t.Error("expected error for unknown startup check mode")
	}
}
