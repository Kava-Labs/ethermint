package types_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
)

func defaultGenesisAccount() types.GenesisAccount {
	return types.GenesisAccount{
		Address: common.BytesToAddress([]byte{0x01}).String(),
		Code:    common.Bytes2Hex([]byte{0x01, 0x02, 0x03}),
		Storage: types.Storage{},
	}
}

func TestGenesisAccountValidate(t *testing.T) {
	testCases := []struct {
		name        string
		getAccount  func() types.GenesisAccount
		expectedErr string
	}{
		{
			name: "default is valid",
			getAccount: func() types.GenesisAccount {
				return defaultGenesisAccount()
			},
			expectedErr: "",
		},
		{
			name: "invalid empty address",
			getAccount: func() types.GenesisAccount {
				account := defaultGenesisAccount()

				account.Address = ""

				return account
			},
			expectedErr: "invalid address",
		},
		{
			name: "invalid address length",
			getAccount: func() types.GenesisAccount {
				account := defaultGenesisAccount()

				account.Address = account.Address[:len(account.Address)-1]

				return account
			},
			expectedErr: "invalid address",
		},
		{
			name: "invalid empty storage key",
			getAccount: func() types.GenesisAccount {
				account := defaultGenesisAccount()

				account.Storage = append(account.Storage, types.State{
					Key: "",
				})

				return account
			},
			expectedErr: "state key hash cannot be blank",
		},
		{
			name: "valid with set storage state",
			getAccount: func() types.GenesisAccount {
				account := defaultGenesisAccount()

				account.Storage = append(account.Storage, types.State{
					Key:   common.BytesToHash([]byte{0x01}).String(),
					Value: common.BytesToHash([]byte{0x02}).String(),
				})

				return account
			},
			expectedErr: "",
		},
		{
			name: "valid with empty code",
			getAccount: func() types.GenesisAccount {
				account := defaultGenesisAccount()

				account.Code = ""

				return account
			},
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.getAccount().Validate()

			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestGenesisStateValidate(t *testing.T) {
	testCases := []struct {
		name        string
		getState    func() *types.GenesisState
		expectedErr string
	}{
		{
			name: "default state is valid",
			getState: func() *types.GenesisState {
				return types.DefaultGenesisState()
			},
			expectedErr: "",
		},
		{
			name: "valid genesis with genesis accounts of same and different state",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				state.Accounts = append(state.Accounts,
					types.GenesisAccount{
						Address: common.BytesToAddress([]byte{0x01}).String(),
						Code:    common.Bytes2Hex([]byte{0x01, 0x02, 0x03}),
						Storage: types.Storage{
							types.State{
								Key:   common.BytesToHash([]byte{0x01}).String(),
								Value: common.BytesToHash([]byte{0x02}).String(),
							},
							types.State{
								Key:   common.BytesToHash([]byte{0x02}).String(),
								Value: common.BytesToHash([]byte{0x03}).String(),
							},
						},
					},
					types.GenesisAccount{
						Address: common.BytesToAddress([]byte{0x02}).String(),
						Code:    common.Bytes2Hex([]byte{0x01, 0x02, 0x03}),
						Storage: types.Storage{
							types.State{
								Key:   common.BytesToHash([]byte{0x01}).String(),
								Value: common.BytesToHash([]byte{0x02}).String(),
							},
							types.State{
								Key:   common.BytesToHash([]byte{0x02}).String(),
								Value: common.BytesToHash([]byte{0x03}).String(),
							},
						},
					},
					types.GenesisAccount{
						Address: common.BytesToAddress([]byte{0x03}).String(),
						Code:    common.Bytes2Hex([]byte{0x04, 0x05, 0x06}),
						Storage: types.Storage{
							types.State{
								Key:   common.BytesToHash([]byte{0x03}).String(),
								Value: common.BytesToHash([]byte{0x04}).String(),
							},
							types.State{
								Key:   common.BytesToHash([]byte{0x05}).String(),
								Value: common.BytesToHash([]byte{0x06}).String(),
							},
						},
					},
				)

				return state
			},
			expectedErr: "",
		},
		{
			name: "empty genesis does not contain valid parameters",
			getState: func() *types.GenesisState {
				return &types.GenesisState{}
			},
			expectedErr: "invalid params",
		},
		{
			name: "default parameters and no accounts is valid",
			getState: func() *types.GenesisState {
				state := &types.GenesisState{}
				state.Params = types.DefaultParams()
				return state
			},
			expectedErr: "",
		},
		{
			name: "genesis is invalid with an invalid genesis account address",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				account := defaultGenesisAccount()
				account.Address = "0x...."

				state.Accounts = append(state.Accounts, account)

				return state
			},
			expectedErr: "invalid address",
		},
		{
			name: "genesis is invalid with an invalid genesis account storage key",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				account := defaultGenesisAccount()
				account.Storage = append(account.Storage, types.State{
					Key:   "", // invalid empty key
					Value: "",
				})

				state.Accounts = append(state.Accounts, account)

				return state
			},
			expectedErr: "invalid storage state",
		},
		{
			name: "genesis is invalid with a genesis account that has duplicated storage keys",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				account := defaultGenesisAccount()
				account.Storage = append(account.Storage,
					types.State{
						Key:   common.BytesToHash([]byte{0x01}).String(),
						Value: "val1",
					},
					types.State{
						Key:   common.BytesToHash([]byte{0x01}).String(),
						Value: "val2",
					},
				)

				state.Accounts = append(state.Accounts, account)

				return state
			},
			expectedErr: "duplicate state key",
		},
		{
			name: "genesis is invalid with a duplicate genesis account",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				account1 := defaultGenesisAccount()
				account2 := defaultGenesisAccount()

				state.Accounts = append(state.Accounts, account1, account2)

				return state
			},
			expectedErr: "duplicated genesis account",
		},
		{
			name: "genesis account validation is checked before uniqueness",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				account1 := defaultGenesisAccount()
				account1.Address = ""

				account2 := defaultGenesisAccount()
				account2.Address = ""

				state.Accounts = append(state.Accounts, account1, account2)

				return state
			},
			expectedErr: "invalid genesis account",
		},
		{
			name: "genesis is invalid if evm denom is invalid",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				state.Params.EvmDenom = "@@@@"

				return state
			},
			expectedErr: "invalid denom",
		},
		{
			name: "genesis is invalid if enabled precompile is invalid",
			getState: func() *types.GenesisState {
				state := types.DefaultGenesisState()

				state.Params.EnabledPrecompiles = append(state.Params.EnabledPrecompiles, "0x....")

				return state
			},
			expectedErr: "invalid hex address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.getState().Validate()

			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}
