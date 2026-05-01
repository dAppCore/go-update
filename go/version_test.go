package updater

import . "dappco.re/go"

func TestVersion_Value_Configured(t *T) {
	AssertNotEqual(t, "", Version)
	AssertEqual(t, PkgVersion, Version)
	AssertContains(t, Version, ".")
}
