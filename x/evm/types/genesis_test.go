package types_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

type GenesisTestSuite struct {
	suite.Suite

	address string
	hash    common.Hash
	code    string
}

func (suite *GenesisTestSuite) SetupTest() {
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes()).String()
	suite.hash = common.BytesToHash([]byte("hash"))
	suite.code = common.Bytes2Hex([]byte{1, 2, 3})
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: types.Storage{
							{Key: suite.hash.String()},
						},
					},
				},
				Params: types.DefaultParams(),
			},
			expPass: true,
		},
		{
			name:     "empty genesis",
			genState: &types.GenesisState{},
			expPass:  false,
		},
		{
			name:     "copied genesis",
			genState: types.NewGenesisState(types.DefaultGenesisState().Params, types.DefaultGenesisState().Accounts),
			expPass:  true,
		},
		{
			name: "invalid genesis",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: common.Address{}.String(),
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid genesis account",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: "123456",

						Code: suite.code,
						Storage: types.Storage{
							{Key: suite.hash.String()},
						},
					},
				},
				Params: types.DefaultParams(),
			},
			expPass: false,
		},
		{
			name: "duplicated genesis account",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: types.Storage{
							types.NewState(suite.hash, suite.hash),
						},
					},
					{
						Address: suite.address,

						Code: suite.code,
						Storage: types.Storage{
							types.NewState(suite.hash, suite.hash),
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "duplicated tx log",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: types.Storage{
							{Key: suite.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid tx log",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: types.Storage{
							{Key: suite.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid params",
			genState: &types.GenesisState{
				Params: types.Params{},
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}
