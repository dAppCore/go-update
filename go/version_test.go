package updater

import . "github.com/dappcore/go"

func TestVersion_Value_Configured(t *T) {
	AssertNotEqual(t, "", Version)
	AssertEqual(t, PkgVersion, Version)
	AssertContains(t, Version, ".")
}
