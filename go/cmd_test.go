package updater

import (
	. "dappco.re/go"
	"github.com/spf13/cobra"
)

func TestCmd_AddUpdateCommands_Good(t *T) {
	root := &cobra.Command{Use: "core"}
	AddUpdateCommands(root)
	cmd, _, err := root.Find([]string{"update"})

	AssertNoError(t, err)
	AssertEqual(t, "update", cmd.Use)
	AssertNotNil(t, cmd.Flags().Lookup("check"))
}

func TestCmd_AddUpdateCommands_Bad(t *T) {
	var root *cobra.Command

	AssertPanics(t, func() {
		AddUpdateCommands(root)
	})
}

func TestCmd_AddUpdateCommands_Ugly(t *T) {
	root := &cobra.Command{Use: "core"}
	AddUpdateCommands(root)
	cmd, _, err := root.Find([]string{"update"})

	AssertNoError(t, err)
	flag := cmd.Flags().Lookup("watch-pid")
	AssertNotNil(t, flag)
	AssertTrue(t, flag.Hidden)
}
