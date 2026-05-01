package updater

import (
	"context"
	"runtime"

	core "dappco.re/go"
	"github.com/spf13/cobra"
)

// Repository configuration for updates
const (
	repoOwner               = "core"
	repoName                = "cli"
	restartWatcherWarning   = "! Could not spawn restart watcher: %v\n"
	failedToApplyUpdate     = "failed to apply update"
	restartingUpdateMessage = "-> Restarting..."
)

// Command flags
var (
	updateChannel  string
	updateForce    bool
	updateCheck    bool
	updateWatchPID int
)

// AddUpdateCommands registers the update command and subcommands.
func AddUpdateCommands(root *cobra.Command) {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update core CLI to the latest version",
		Long: `Update the core CLI to the latest version from GitHub releases.

By default, checks the 'stable' channel for tagged releases (v*.*.*)
Use --channel=dev for the latest development build.

Examples:
  core update              # Update to latest stable release
  core update --check      # Check for updates without applying
  core update --channel=dev   # Update to latest dev build
  core update --force      # Force update even if already on latest`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := runUpdate(cmd, args)
			if r.OK {
				return nil
			}
			return core.NewError(r.Error())
		},
	}

	updateCmd.PersistentFlags().StringVar(&updateChannel, "channel", "stable", "Release channel: stable, beta, alpha, prerelease, or dev")
	updateCmd.PersistentFlags().BoolVar(&updateForce, "force", false, "Force update even if already on latest version")
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Only check for updates, do not apply")
	updateCmd.Flags().IntVar(&updateWatchPID, "watch-pid", 0, "Internal: watch for parent PID to die then restart")
	if err := updateCmd.Flags().MarkHidden("watch-pid"); err != nil {
		panic(err)
	}

	updateCmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Check for available updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			previousCheck := updateCheck
			updateCheck = true
			defer func() { updateCheck = previousCheck }()
			r := runUpdate(cmd, args)
			if r.OK {
				return nil
			}
			return core.NewError(r.Error())
		},
	})

	root.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) core.Result {
	// If we're in watch mode, wait for parent to die then restart
	if updateWatchPID > 0 {
		return watchAndRestart(updateWatchPID)
	}

	currentVersion := Version
	normalizedChannel := normaliseGitHubChannel(updateChannel)

	core.Print(nil, "Current version: %s", currentVersion)
	core.Print(nil, "Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	core.Print(nil, "Channel: %s", normalizedChannel)

	// Handle dev channel specially - it's a prerelease tag, not a semver channel
	if normalizedChannel == "dev" {
		return handleDevUpdate(currentVersion)
	}

	// Check for newer version
	check := CheckForNewerVersion(repoOwner, repoName, normalizedChannel, true)
	if !check.OK {
		return updateCommandError(core.NewError(check.Error()), "failed to check for updates")
	}
	state := check.Value.(versionCheck)

	if state.release == nil {
		core.Print(nil, "! No releases found in %s channel", normalizedChannel)
		return core.Ok(nil)
	}

	if !state.updateAvailable && !updateForce {
		core.Print(nil, "OK Already on latest version (%s)", state.release.TagName)
		return core.Ok(nil)
	}

	core.Print(nil, "Latest version: %s", state.release.TagName)

	if updateCheck {
		if state.updateAvailable {
			core.Print(nil, "! Update available: %s -> %s",
				currentVersion,
				state.release.TagName)
			core.Println("Run core update to update")
		}
		return core.Ok(nil)
	}

	// Spawn watcher before applying update
	if r := spawnWatcher(); !r.OK {
		// If watcher fails, continue anyway - update will still work
		core.Print(nil, restartWatcherWarning, r.Error())
	}

	// Apply update
	core.Println("-> Downloading update...")

	downloadURL := GetDownloadURL(state.release, "")
	if !downloadURL.OK {
		return updateCommandError(core.NewError(downloadURL.Error()), "failed to get download URL")
	}

	if r := DoUpdate(downloadURL.Value.(string)); !r.OK {
		return updateCommandError(core.NewError(r.Error()), failedToApplyUpdate)
	}

	core.Print(nil, "OK Updated to %s", state.release.TagName)
	core.Println(restartingUpdateMessage)

	return core.Ok(nil)
}

// handleDevUpdate handles updates from the dev release (rolling prerelease)
func handleDevUpdate(currentVersion string) core.Result {
	client := NewGithubClient()

	// Fetch the dev release directly by tag
	result := client.GetLatestRelease(context.Background(), repoOwner, repoName, "beta")
	if !result.OK {
		// Try fetching the "dev" tag directly
		return handleDevTagUpdate(currentVersion)
	}
	release := result.Value.(*Release)

	if release == nil {
		return handleDevTagUpdate(currentVersion)
	}

	core.Print(nil, "Latest dev: %s", release.TagName)

	if updateCheck {
		core.Println("Run core update --channel=dev to update")
		return core.Ok(nil)
	}

	// Spawn watcher before applying update
	if r := spawnWatcher(); !r.OK {
		core.Print(nil, restartWatcherWarning, r.Error())
	}

	core.Println("-> Downloading update...")

	downloadURL := GetDownloadURL(release, "")
	if !downloadURL.OK {
		return updateCommandError(core.NewError(downloadURL.Error()), "failed to get download URL")
	}

	if r := DoUpdate(downloadURL.Value.(string)); !r.OK {
		return updateCommandError(core.NewError(r.Error()), failedToApplyUpdate)
	}

	core.Print(nil, "OK Updated to %s", release.TagName)
	core.Println(restartingUpdateMessage)

	return core.Ok(nil)
}

// handleDevTagUpdate fetches the dev release using the direct tag
func handleDevTagUpdate(currentVersion string) core.Result {
	// Construct download URL directly for dev release
	downloadURL := core.Sprintf(
		"https://github.com/%s/%s/releases/download/dev/core-%s-%s",
		repoOwner, repoName, runtime.GOOS, runtime.GOARCH,
	)

	if runtime.GOOS == "windows" {
		downloadURL += ".exe"
	}

	core.Println("Latest: dev (rolling)")

	if updateCheck {
		core.Println("Run core update --channel=dev to update")
		return core.Ok(nil)
	}

	// Spawn watcher before applying update
	if r := spawnWatcher(); !r.OK {
		core.Print(nil, restartWatcherWarning, r.Error())
	}

	core.Println("-> Downloading from dev release...")

	if r := DoUpdate(downloadURL); !r.OK {
		return updateCommandError(core.NewError(r.Error()), failedToApplyUpdate)
	}

	core.Println("OK Updated to latest dev build")
	core.Println(restartingUpdateMessage)

	return core.Ok(nil)
}

func updateCommandError(err error, msg string) core.Result {
	return core.Fail(core.Wrap(err, "update.command", msg))
}
