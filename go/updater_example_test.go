package updater

import . "github.com/dappcore/go"

func ExampleCheckOnly() {
	Println(formatVersionForDisplay("1.2.3", true))
}
