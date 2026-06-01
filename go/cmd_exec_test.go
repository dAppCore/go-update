package updater

import (
	. "dappco.re/go"
	"github.com/spf13/cobra"
)

// runCmdArgs builds a fresh root with the update commands wired, sets the
// args and executes, returning the error. The package flag vars and seams are
// restored by the deferred fixture in each caller.
func runCmdArgs(args ...string) error {
	root := &cobra.Command{Use: "core", SilenceUsage: true, SilenceErrors: true}
	AddUpdateCommands(root)
	root.SetArgs(args)
	return root.Execute()
}

func TestCmdExec_Update_Check_Good(t *T) {
	// `update --check` with an available update runs the RunE closure, reports,
	// and returns nil without applying.
	defer cmdRunFixture()()
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{TagName: "v1.2.0"}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	err := runCmdArgs("update", "--check")

	AssertNoError(t, err)
	AssertLen(t, applied, 0)
}

func TestCmdExec_Update_Check_Bad(t *T) {
	// The RunE closure converts a failing check into a returned error.
	defer cmdRunFixture()()
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latestErr: "github down"}
	}

	err := runCmdArgs("update", "--check")

	AssertError(t, err)
	AssertContains(t, err.Error(), "failed to check for updates")
}

func TestCmdExec_Update_CheckSubcommand_Ugly(t *T) {
	// The `update check` subcommand forces check mode through its own RunE and
	// restores the previous flag afterwards.
	defer cmdRunFixture()()
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latest: &Release{TagName: "v1.2.0"}}
	}
	applied := []string{}
	DoUpdate = func(url string) Result {
		applied = append(applied, url)
		return Ok(nil)
	}

	err := runCmdArgs("update", "check")

	AssertNoError(t, err)
	AssertLen(t, applied, 0)
}

func TestCmdExec_Update_CheckSubcommand_Bad(t *T) {
	defer cmdRunFixture()()
	Version = "1.0.0"
	NewGithubClient = func() GithubClient {
		return flowsTestClient{latestErr: "rate limit"}
	}

	err := runCmdArgs("update", "check")

	AssertError(t, err)
	AssertContains(t, err.Error(), "failed to check for updates")
}
