package eip712

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedDataDomain(t *testing.T) {
	domain := getTypedDataDomain(1234)

	domainMap := domain.Map()

	// Verify both len and expected contents in order to assert that no other
	// fields are present
	require.Len(t, domainMap, 3)
	require.Contains(t, domainMap, "chainId")
	require.Contains(t, domainMap, "name")
	require.Contains(t, domainMap, "version")

	// Extra check to ensure that the fields that are not used for signature
	// verification are not present in the map. Should be in conjunction with
	// the checks above to ensure there isn't a different variant of these
	// fields present, e.g. different casing.
	require.NotContains(t, domainMap, "verifyingContract")
	require.NotContains(t, domainMap, "salt")
}
