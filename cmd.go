package updater

import (
	"context"
	"fmt"
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
		RunE: runUpdate,
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
			return runUpdate(cmd, args)
		},
	})

	root.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// If we're in watch mode, wait for parent to die then restart
	if updateWatchPID > 0 {
		return watchAndRestart(updateWatchPID)
	}

	currentVersion := Version
	normalizedChannel := normaliseGitHubChannel(updateChannel)

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Channel: %s\n\n", normalizedChannel)

	// Handle dev channel specially - it's a prerelease tag, not a semver channel
	if normalizedChannel == "dev" {
		return handleDevUpdate(currentVersion)
	}

	// Check for newer version
	release, updateAvailable, err := CheckForNewerVersion(repoOwner, repoName, normalizedChannel, true)
	if err != nil {
		return updateCommandError(err, "failed to check for updates")
	}

	if release == nil {
		fmt.Printf("! No releases found in %s channel\n", normalizedChannel)
		return nil
	}

	if !updateAvailable && !updateForce {
		fmt.Printf("OK Already on latest version (%s)\n", release.TagName)
		return nil
	}

	fmt.Printf("Latest version: %s\n", release.TagName)

	if updateCheck {
		if updateAvailable {
			fmt.Printf("\n! Update available: %s -> %s\n",
				currentVersion,
				release.TagName)
			fmt.Println("Run core update to update")
		}
		return nil
	}

	// Spawn watcher before applying update
	if err := spawnWatcher(); err != nil {
		// If watcher fails, continue anyway - update will still work
		fmt.Printf(restartWatcherWarning, err)
	}

	// Apply update
	fmt.Println("\n-> Downloading update...")

	downloadURL, err := GetDownloadURL(release, "")
	if err != nil {
		return updateCommandError(err, "failed to get download URL")
	}

	if err := DoUpdate(downloadURL); err != nil {
		return updateCommandError(err, failedToApplyUpdate)
	}

	fmt.Printf("OK Updated to %s\n", release.TagName)
	fmt.Println(restartingUpdateMessage)

	return nil
}

// handleDevUpdate handles updates from the dev release (rolling prerelease)
func handleDevUpdate(currentVersion string) error {
	client := NewGithubClient()

	// Fetch the dev release directly by tag
	release, err := client.GetLatestRelease(context.Background(), repoOwner, repoName, "beta")
	if err != nil {
		// Try fetching the "dev" tag directly
		return handleDevTagUpdate(currentVersion)
	}

	if release == nil {
		return handleDevTagUpdate(currentVersion)
	}

	fmt.Printf("Latest dev: %s\n", release.TagName)

	if updateCheck {
		fmt.Println("\nRun core update --channel=dev to update")
		return nil
	}

	// Spawn watcher before applying update
	if err := spawnWatcher(); err != nil {
		fmt.Printf(restartWatcherWarning, err)
	}

	fmt.Println("\n-> Downloading update...")

	downloadURL, err := GetDownloadURL(release, "")
	if err != nil {
		return updateCommandError(err, "failed to get download URL")
	}

	if err := DoUpdate(downloadURL); err != nil {
		return updateCommandError(err, failedToApplyUpdate)
	}

	fmt.Printf("OK Updated to %s\n", release.TagName)
	fmt.Println(restartingUpdateMessage)

	return nil
}

// handleDevTagUpdate fetches the dev release using the direct tag
func handleDevTagUpdate(currentVersion string) error {
	// Construct download URL directly for dev release
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/dev/core-%s-%s",
		repoOwner, repoName, runtime.GOOS, runtime.GOARCH,
	)

	if runtime.GOOS == "windows" {
		downloadURL += ".exe"
	}

	fmt.Println("Latest: dev (rolling)")

	if updateCheck {
		fmt.Println("\nRun core update --channel=dev to update")
		return nil
	}

	// Spawn watcher before applying update
	if err := spawnWatcher(); err != nil {
		fmt.Printf(restartWatcherWarning, err)
	}

	fmt.Println("\n-> Downloading from dev release...")

	if err := DoUpdate(downloadURL); err != nil {
		return updateCommandError(err, failedToApplyUpdate)
	}

	fmt.Println("OK Updated to latest dev build")
	fmt.Println(restartingUpdateMessage)

	return nil
}

func updateCommandError(err error, msg string) error {
	return core.Wrap(err, "update.command", msg)
}
