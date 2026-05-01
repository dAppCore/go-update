package updater

import . "dappco.re/go"

func ExampleCheckOnly() {
	Println(formatVersionForDisplay("1.2.3", true))
}
