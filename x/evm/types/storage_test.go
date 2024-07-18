package types_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
)

var (
	validState1 = types.NewState(common.BytesToHash([]byte{1}), common.BytesToHash([]byte{1, 1}))
	validState2 = types.NewState(common.BytesToHash([]byte{2}), common.BytesToHash([]byte{2, 1}))
	validState3 = types.NewState(common.BytesToHash([]byte{3}), common.BytesToHash([]byte{3, 1}))

	invalidStateEmptyKey          = types.State{"", common.BytesToHash([]byte{1, 1, 1}).String()}
	invalidStateEmptyKeyWithSpace = types.State{" ", common.BytesToHash([]byte{1, 1, 2}).String()}

	validStateEmptyValue          = types.State{common.BytesToHash([]byte{4}).String(), ""}
	validStateEmptyValueWithSpace = types.State{common.BytesToHash([]byte{5}).String(), " "}
)

func TestStorageValidate(t *testing.T) {
	testCases := []struct {
		name        string
		storage     types.Storage
		expectedErr string
	}{
		{
			name:        "valid empty storage",
			storage:     types.Storage{},
			expectedErr: "",
		},
		{
			name: "valid collection of states",
			storage: types.Storage{
				validState1,
				validState2,
				validState3,
			},
			expectedErr: "",
		},
		{
			name: "valid collection of states any order",
			storage: types.Storage{
				validState2,
				validState1,
				validState3,
			},
			expectedErr: "",
		},
		{
			name: "duplicate state key in middle",
			storage: types.Storage{
				validState1,
				validState2,
				validState1,
				validState3,
			},
			expectedErr: "duplicate state key",
		},
		{
			name: "duplicate state key in last element",
			storage: types.Storage{
				validState1,
				validState2,
				validState3,
				validState3,
			},
			expectedErr: "duplicate state key",
		},
		{
			name: "invalid empty state key",
			storage: types.Storage{
				validState1,
				invalidStateEmptyKey,
				validState2,
			},
			expectedErr: "state key hash cannot be blank",
		},
		{
			name: "invalid empty state key only key",
			storage: types.Storage{
				invalidStateEmptyKey,
			},
			expectedErr: "state key hash cannot be blank",
		},
		{
			name: "invalid state key only whitespace",
			storage: types.Storage{
				invalidStateEmptyKeyWithSpace,
			},
			expectedErr: "state key hash cannot be blank",
		},
		{
			name: "valid state with empty value",
			storage: types.Storage{
				validStateEmptyValue,
				validStateEmptyValueWithSpace,
			},
			expectedErr: "",
		},
		{
			name: "state validity is checked before state key uniqueness",
			storage: types.Storage{
				invalidStateEmptyKey,
				invalidStateEmptyKey,
			},
			expectedErr: "state key hash cannot be blank",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.storage.Validate()
			if tc.expectedErr == "" {
				require.NoError(t, err, tc.name)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestStorageString(t *testing.T) {
	storage := types.Storage{types.NewState(common.BytesToHash([]byte("key")), common.BytesToHash([]byte("value")))}
	str := "key:\"0x00000000000000000000000000000000000000000000000000000000006b6579\" value:\"0x00000000000000000000000000000000000000000000000000000076616c7565\" \n"
	require.Equal(t, str, storage.String())
}
