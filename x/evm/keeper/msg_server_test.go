package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
)

func (suite *KeeperTestSuite) TestEthereumTx() {
	var (
		err             error
		msg             *types.MsgEthereumTx
		signer          ethtypes.Signer
		vmdb            *statedb.StateDB
		chainCfg        *params.ChainConfig
		expectedGasUsed uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   bool
	}{
		{
			"Deploy contract tx - insufficient gas",
			func() {
				msg, err = suite.createContractMsgTx(
					vmdb.GetNonce(suite.address),
					signer,
					chainCfg,
					big.NewInt(1),
				)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"Transfer funds tx",
			func() {
				msg, _, err = newEthMsgTx(
					vmdb.GetNonce(suite.address),
					suite.ctx.BlockHeight(),
					suite.address,
					chainCfg,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
				expectedGasUsed = params.TxGas
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			keeperParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
			vmdb = suite.StateDB()

			tc.malleate()
			res, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, msg)
			if tc.expErr {
				suite.Require().Error(err)
				return
			}
			suite.Require().NoError(err)
			suite.Require().Equal(expectedGasUsed, res.GasUsed)
			suite.Require().False(res.Failed())
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name      string
		request   *types.MsgUpdateParams
		expectErr bool
	}{
		{
			name:      "fail - invalid authority",
			request:   &types.MsgUpdateParams{Authority: "foobar"},
			expectErr: true,
		},
		{
			name: "pass - valid Update msg",
			request: &types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    types.DefaultParams(),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run("MsgUpdateParams", func() {
			_, err := suite.app.EvmKeeper.UpdateParams(suite.ctx, tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdatePrecompiles() {
	addr1 := "0x1000000000000000000000000000000000000000"
	addr2 := "0x2000000000000000000000000000000000000000"
	addr3 := "0x3000000000000000000000000000000000000000"

	testCases := []struct {
		name               string
		enabledPrecompiles []string
		// precompiles which must be uninitialized after corresponding test case
		uninitialized []string
	}{
		{
			name:               "enable addr1 and addr2",
			enabledPrecompiles: []string{addr1, addr2},
			uninitialized:      []string{addr3},
		},
		{
			name:               "enable addr3, and disable the rest",
			enabledPrecompiles: []string{addr3},
			uninitialized:      []string{addr1, addr2},
		},
		{
			name:               "no changes",
			enabledPrecompiles: []string{addr3},
			uninitialized:      []string{addr1, addr2},
		},
		{
			name:               "enable all precompiles",
			enabledPrecompiles: []string{addr1, addr2, addr3},
			uninitialized:      []string{},
		},
		{
			name:               "disable all precompiles",
			enabledPrecompiles: []string{},
			uninitialized:      []string{addr1, addr2, addr3},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := suite.app.EvmKeeper.GetParams(suite.ctx)
			params.EnabledPrecompiles = tc.enabledPrecompiles

			_, err := suite.app.EvmKeeper.UpdateParams(sdk.WrapSDKContext(suite.ctx), &types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    params,
			})
			suite.Require().NoError(err)

			vmdb := suite.StateDB()

			// check that precompiles are initialized
			for _, hexAddr := range tc.enabledPrecompiles {
				addr := common.HexToAddress(hexAddr)

				// A precompile address must exist and be non-empty
				suite.True(vmdb.Exist(addr), "expected enabled precompile %s to exist", hexAddr)
				suite.False(vmdb.Empty(addr), "expected enabled precompile %s to not be empty", hexAddr)

				// A precompile address must have nonce 1, code set to 0x01, and have a byte length of 1
				suite.Equal(keeper.PrecompileNonce, vmdb.GetNonce(addr), "expected enabled precompile %s to have nonce set in state", hexAddr)
				suite.Equal(keeper.PrecompileCode, vmdb.GetCode(addr), "expected enabled precompile %s to have code set in state", hexAddr)
				suite.Equal(1, vmdb.GetCodeSize(addr), "expected enabled precompile %s to have code size of 1 byte", hexAddr)
			}

			// check that precompiles are uninitialized
			for _, hexAddr := range tc.uninitialized {
				addr := common.HexToAddress(hexAddr)

				// A precompile address must not exist
				suite.False(vmdb.Exist(addr), "expected uninitialized precompile %s to not exist", hexAddr)
				suite.Require().Equal(uint64(0), vmdb.GetNonce(addr))
				suite.Require().Equal([]byte(nil), vmdb.GetCode(addr))
			}
		})
	}
}
