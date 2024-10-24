package keeper_test

import (
	"reflect"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	legacytypes "github.com/evmos/ethermint/x/evm/types/legacy"
	legacytestutil "github.com/evmos/ethermint/x/evm/types/legacy/testutil"
)

func (suite *KeeperTestSuite) TestParams() {
	addr1 := "0x1000000000000000000000000000000000000000"
	addr2 := "0x2000000000000000000000000000000000000000"

	params := suite.app.EvmKeeper.GetParams(suite.ctx)
	err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			"success - Checks if the default params are set correctly",
			func() interface{} {
				return types.DefaultParams()
			},
			func() interface{} {
				return suite.app.EvmKeeper.GetParams(suite.ctx)
			},
			true,
		},
		{
			"success - EvmDenom param is set to \"inj\" and can be retrieved correctly",
			func() interface{} {
				params.EvmDenom = "inj"
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EvmDenom
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEvmDenom()
			},
			true,
		},
		{
			"success - Check EnableCreate param is set to false and can be retrieved correctly",
			func() interface{} {
				params.EnableCreate = false
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EnableCreate
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnableCreate()
			},
			true,
		},
		{
			"success - Check EnableCall param is set to false and can be retrieved correctly",
			func() interface{} {
				params.EnableCall = false
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EnableCall
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnableCall()
			},
			true,
		},
		{
			"success - Check AllowUnprotectedTxs param is set to false and can be retrieved correctly",
			func() interface{} {
				params.AllowUnprotectedTxs = false
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.AllowUnprotectedTxs
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetAllowUnprotectedTxs()
			},
			true,
		},
		{
			"success - Check ChainConfig param is set to the default value and can be retrieved correctly",
			func() interface{} {
				params.ChainConfig = types.DefaultChainConfig()
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.ChainConfig
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetChainConfig()
			},
			true,
		},
		{
			"success - EnabledPrecompiles param is set to empty slice and will be retrieved as nil",
			func() interface{} {
				params.EnabledPrecompiles = []string{}
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				var typedNil []string = nil // NOTE: despite we set EnabledPrecompiles as []string{}, it will be retrieved as nil
				return typedNil
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnabledPrecompiles()
			},
			true,
		},
		{
			"success - EnabledPrecompiles param is set to nil and can be retrieved correctly",
			func() interface{} {
				params.EnabledPrecompiles = nil
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EnabledPrecompiles
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnabledPrecompiles()
			},
			true,
		},
		{
			"success - EnabledPrecompiles param is set to []string{addr1, addr2} and can be retrieved correctly",
			func() interface{} {
				params.EnabledPrecompiles = []string{addr1, addr2}
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EnabledPrecompiles
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnabledPrecompiles()
			},
			true,
		},
		{
			"failure - EnabledPrecompiles param is set to []string{addr2, addr1} which fails is_sorted validation",
			func() interface{} {
				params.EnabledPrecompiles = []string{addr2, addr1}
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().Error(err)
				return params.EnabledPrecompiles
			},
			func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnabledPrecompiles()
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			suite.Require().Equal(tc.expected, outcome)
		})
	}
}

func (suite *KeeperTestSuite) TestLegacyParamsKeyTableRegistration() {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec
	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})
	ak := suite.app.AccountKeeper

	// paramspace used only for setting legacy parameters (not given to keeper)
	setParamSpace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	).WithKeyTable(legacytypes.ParamKeyTable())
	params := legacytypes.DefaultParams()
	params.EIP712AllowedMsgs = legacytestutil.TestEIP712AllowedMsgs
	setParamSpace.SetParamSet(ctx, &params)

	// param space that has not been created with a key table
	unregisteredSubspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	)

	// assertion required to ensure we are testing correctness
	// of a keeper receiving a subpsace without a key table registration
	suite.Require().False(unregisteredSubspace.HasKeyTable())

	newKeeper := func() *keeper.Keeper {
		// create a keeper, mimicking an app.go which has not registered the key table
		return keeper.NewKeeper(
			cdc, storeKey, tKey, authtypes.NewModuleAddress("gov"),
			ak,
			nil, nil, nil, // OK to pass nil in for these since we only instantiate and use params
			vm.NewEVM,
			"",
			unregisteredSubspace,
		)
	}
	k := newKeeper()

	// the keeper must set the key table
	var fetchedParams types.Params
	suite.Require().NotPanics(func() { fetchedParams = k.GetParams(ctx) })
	// this modifies the internal data of the subspace, so we should see the key table registered
	suite.Require().True(unregisteredSubspace.HasKeyTable())
	// ensure returned params are equal to the set legacy parameters
	legacytestutil.AssertParamsEqual(suite.T(), params, fetchedParams)
	// ensure we do not attempt to override any existing key tables to keep compatibility
	// when passing a subpsace to the keeper that has already been used to work with parameters
	suite.Require().NotPanics(func() { newKeeper() })
}

func (suite *KeeperTestSuite) TestRenamedFieldReturnsProperValueForLegacyParams() {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec
	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})
	ak := suite.app.AccountKeeper

	// paramspace used only for setting legacy parameters (not given to keeper)
	legacyParamstore := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	).WithKeyTable(legacytypes.ParamKeyTable())

	oldParams := legacytypes.DefaultParams()
	// ensure this is set regardless of default param refactoring
	mergeBlock := sdkmath.NewInt(9999)
	oldParams.ChainConfig.MergeForkBlock = &mergeBlock
	// set legacy params with merge block set
	legacyParamstore.SetParamSet(ctx, &oldParams)

	// new subspace for keeper, mimicking what a new binary would do
	subspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	)
	k := keeper.NewKeeper(
		cdc, storeKey, tKey, authtypes.NewModuleAddress("gov"),
		ak,
		nil, nil, nil,
		vm.NewEVM,
		"",
		subspace,
	)

	params := k.GetParams(ctx)

	suite.Require().Equal(params.ChainConfig.MergeNetsplitBlock, oldParams.ChainConfig.MergeForkBlock)
}

func (suite *KeeperTestSuite) TestNilLegacyParamsDoNotPanic() {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec
	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})
	ak := suite.app.AccountKeeper

	subspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	)

	k := keeper.NewKeeper(
		cdc, storeKey, tKey, authtypes.NewModuleAddress("gov"),
		ak,
		nil, nil, nil, // OK to pass nil in for these since we only instantiate and use params
		vm.NewEVM,
		"",
		subspace,
	)

	suite.Require().NotPanics(func() { k.GetParams(ctx) })
}
