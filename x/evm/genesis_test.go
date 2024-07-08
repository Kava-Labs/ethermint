package evm_test

import (
	"math/big"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	precompile_modules "github.com/ethereum/go-ethereum/precompile/modules"

	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	etherminttypes "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
)

func (suite *EvmTestSuite) TestInitGenesis() {
	privkey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	address := common.HexToAddress(privkey.PubKey().Address().String())
	hexAddr1 := "0x1000000000000000000000000000000000000000"
	hexAddr2 := "0x2000000000000000000000000000000000000000"

	var vmdb *statedb.StateDB

	testCases := []struct {
		name              string
		malleate          func()
		getGenState       func() *types.GenesisState
		registeredModules []precompile_modules.Module
		expPanic          bool
	}{
		{
			name:     "default",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				return types.DefaultGenesisState()
			},
			expPanic: false,
		},
		{
			name: "valid account",
			malleate: func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			getGenState: func() *types.GenesisState {
				return &types.GenesisState{
					Params: types.DefaultParams(),
					Accounts: []types.GenesisAccount{
						{
							Address: address.String(),
							Storage: types.Storage{
								{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
							},
						},
					},
				}
			},
			expPanic: false,
		},
		{
			name:     "account not found",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				return &types.GenesisState{
					Params: types.DefaultParams(),
					Accounts: []types.GenesisAccount{
						{
							Address: address.String(),
						},
					},
				}
			},
			expPanic: true,
		},
		{
			name: "invalid account type",
			malleate: func() {
				acc := authtypes.NewBaseAccountWithAddress(address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			getGenState: func() *types.GenesisState {
				return &types.GenesisState{
					Params: types.DefaultParams(),
					Accounts: []types.GenesisAccount{
						{
							Address: address.String(),
						},
					},
				}
			},
			expPanic: true,
		},
		{
			name: "invalid code hash",
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			getGenState: func() *types.GenesisState {
				return &types.GenesisState{
					Params: types.DefaultParams(),
					Accounts: []types.GenesisAccount{
						{
							Address: address.String(),
							Code:    "ffffffff",
						},
					},
				}
			},
			expPanic: true,
		},
		{
			name: "ignore empty account code checking",
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())

				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			getGenState: func() *types.GenesisState {
				return &types.GenesisState{
					Params: types.DefaultParams(),
					Accounts: []types.GenesisAccount{
						{
							Address: address.String(),
							Code:    "",
						},
					},
				}
			},
			expPanic: false,
		},
		{
			name: "ignore empty account code checking with non-empty codehash",
			malleate: func() {
				ethAcc := &etherminttypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccount(address.Bytes(), nil, 0, 0),
					CodeHash:    common.BytesToHash([]byte{1, 2, 3}).Hex(),
				}

				suite.app.AccountKeeper.SetAccount(suite.ctx, ethAcc)
			},
			getGenState: func() *types.GenesisState {
				return &types.GenesisState{
					Params: types.DefaultParams(),
					Accounts: []types.GenesisAccount{
						{
							Address: address.String(),
							Code:    "",
						},
					},
				}
			},
			expPanic: false,
		},
		{
			name:     "precompile is enabled, but not registered",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1}
				return defaultGen
			},
			registeredModules: nil,
			expPanic:          true,
		},
		{
			name:     "enabled precompiles are not sorted",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr2, hexAddr1}
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
				{Address: common.HexToAddress(hexAddr2)},
			},
			expPanic: true,
		},
		{
			name:     "enabled precompiles are not unique",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1, hexAddr1}
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
			},
			expPanic: true,
		},
		{
			name:     "enabled precompile",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1, hexAddr1}
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
			},
			expPanic: true,
		},
		{
			name:     "precompile is enabled and registered without any account state set",
			malleate: func() {},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1}
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
			},
			expPanic: true,
		},
		{
			name: "precompile is enabled and registered with code set but not nonce",
			malleate: func() {
				addr := common.HexToAddress(hexAddr1).Bytes()

				precompileAcc := &etherminttypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccount(addr, nil, 9999, 0),
					CodeHash:    crypto.Keccak256Hash([]byte{0x1}).Hex(),
				}

				suite.app.AccountKeeper.SetAccount(suite.ctx, precompileAcc)
			},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1}
				defaultGen.Accounts = append(defaultGen.Accounts, types.GenesisAccount{
					Address: hexAddr1,
					Code:    common.Bytes2Hex([]byte{0x1}),
				})
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
			},
			expPanic: true,
		},
		{
			name: "precompile is enabled and registered with nonce set but not code",
			malleate: func() {
				addr := common.HexToAddress(hexAddr1).Bytes()

				precompileAcc := &etherminttypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccount(addr, nil, 9999, 1),
					CodeHash:    crypto.Keccak256Hash([]byte{0x1}).Hex(),
				}

				suite.app.AccountKeeper.SetAccount(suite.ctx, precompileAcc)
			},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1}
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
			},
			expPanic: true,
		},
		{
			name: "precompile is enabled and registered with nonce and code set",
			malleate: func() {
				addr := common.HexToAddress(hexAddr1).Bytes()

				precompileAcc := &etherminttypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccount(addr, nil, 9999, 1),
					CodeHash:    crypto.Keccak256Hash([]byte{0x1}).Hex(),
				}

				suite.app.AccountKeeper.SetAccount(suite.ctx, precompileAcc)
			},
			getGenState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesisState()
				defaultGen.Params.EnabledPrecompiles = []string{hexAddr1}
				defaultGen.Accounts = append(defaultGen.Accounts, types.GenesisAccount{
					Address: hexAddr1,
					Code:    common.Bytes2Hex([]byte{0x1}),
				})
				return defaultGen
			},
			registeredModules: []precompile_modules.Module{
				{Address: common.HexToAddress(hexAddr1)},
			},
			expPanic: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset values
			vmdb = suite.StateDB()

			tc.malleate()
			vmdb.Commit()

			if tc.expPanic {
				suite.Require().Panics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, *tc.getGenState(), tc.registeredModules)
					},
				)
			} else {
				suite.Require().NotPanics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, *tc.getGenState(), tc.registeredModules)
					},
				)
			}
		})
	}
}
