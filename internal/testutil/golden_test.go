package testutil

import "testing"

// testdata/sample.golden is committed with exactly the bytes below; this proves
// the comparator path. The generator's real golden tests (AD-15) will use the
// same helper against emitted container code.
func TestGoldenMatchesCommittedFixture(t *testing.T) {
	Golden(t, "sample", []byte("hello, godi golden\n"))
}
