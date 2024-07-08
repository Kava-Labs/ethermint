package statedb_test

import (
	"testing"

	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/stretchr/testify/require"
)

func TestAccountIsContract(t *testing.T) {
	tests := []struct {
		name     string
		account  statedb.Account
		expected bool
	}{
		{
			name: "Empty account",
			account: statedb.Account{
				Nonce:    0,
				CodeHash: emptyCodeHash,
			},
			expected: false,
		},
		{
			name: "Non-empty code hash",
			account: statedb.Account{
				Nonce:    0,
				CodeHash: []byte{1, 2, 3},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(
				t,
				tt.expected,
				tt.account.IsContract(),
			)
		})
	}
}
