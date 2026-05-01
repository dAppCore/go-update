package updater

import "github.com/spf13/cobra"

func ExampleAddUpdateCommands() {
	root := &cobra.Command{Use: "core"}
	AddUpdateCommands(root)
}
