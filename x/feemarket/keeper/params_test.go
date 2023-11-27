package keeper_test

import (
	"reflect"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/x/feemarket/keeper"
	"github.com/evmos/ethermint/x/feemarket/types"
)

func (suite *KeeperTestSuite) TestGetParams() {
	params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
	suite.Require().NotNil(params.BaseFee)
	suite.Require().NotNil(params.MinGasPrice)
	suite.Require().NotNil(params.MinGasMultiplier)
}

func (suite *KeeperTestSuite) TestSetGetParams() {
	params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
	suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
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
				return suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			},
			true,
		},
		{
			"success - Check ElasticityMultiplier is set to 3 and can be retrieved correctly",
			func() interface{} {
				params.ElasticityMultiplier = 3
				suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				return params.ElasticityMultiplier
			},
			func() interface{} {
				return suite.app.FeeMarketKeeper.GetParams(suite.ctx).ElasticityMultiplier
			},
			true,
		},
		{
			"success - Check BaseFeeEnabled is computed with its default params and can be retrieved correctly",
			func() interface{} {
				suite.app.FeeMarketKeeper.SetParams(suite.ctx, types.DefaultParams())
				return true
			},
			func() interface{} {
				return suite.app.FeeMarketKeeper.GetBaseFeeEnabled(suite.ctx)
			},
			true,
		},
		{
			"success - Check BaseFeeEnabled is computed with alternate params and can be retrieved correctly",
			func() interface{} {
				params.NoBaseFee = true
				params.EnableHeight = 5
				suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				return true
			},
			func() interface{} {
				return suite.app.FeeMarketKeeper.GetBaseFeeEnabled(suite.ctx)
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
	ctx := testutil.DefaultContext(storeKey, tKey)

	// paramspace used only for setting legacy parameters (not given to keeper)
	setParamSpace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		storeKey,
		tKey,
		"feemarket",
	).WithKeyTable(types.ParamKeyTable())
	params := types.DefaultParams()
	setParamSpace.SetParamSet(ctx, &params)

	// param space that has not been created with a key table
	unregisteredSubspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		storeKey,
		tKey,
		"feemarket",
	)

	// assertion required to ensure we are testing correctness
	// of a keeper receiving a subpsace without a key table registration
	suite.Require().False(unregisteredSubspace.HasKeyTable())

	// create a keeper, mimicking an app.go which has not registered the key table
	k := keeper.NewKeeper(cdc, authtypes.NewModuleAddress("gov"), storeKey, tKey, unregisteredSubspace)

	// the keeper must set the key table
	var fetchedParams types.Params
	suite.Require().NotPanics(func() { fetchedParams = k.GetParams(ctx) })
	// this modifies the internal data of the subspace, so we should see the key table registered
	suite.Require().True(unregisteredSubspace.HasKeyTable())
	// general check that params match what we set and are not nil
	suite.Require().Equal(params, fetchedParams)
	// ensure we do not attempt to override any existing key tables to keep compatibility
	// when passing a subpsace to the keeper that has already been used to work with parameters
	suite.Require().NotPanics(func() { keeper.NewKeeper(cdc, authtypes.NewModuleAddress("gov"), storeKey, tKey, unregisteredSubspace) })
}
