package v3_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	v3 "github.com/evmos/ethermint/x/evm/migrations/v3"
	legacytypes "github.com/evmos/ethermint/x/evm/types/legacy"
	legacytestutil "github.com/evmos/ethermint/x/evm/types/legacy/testutil"
)

func TestMigrate(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})
	kvStore := ctx.KVStore(storeKey)

	paramstore := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	).WithKeyTable(legacytypes.ParamKeyTable())

	initialParams := legacytypes.DefaultParams()

	// new params treats an empty slice as nil
	initialParams.EIP712AllowedMsgs = nil

	paramstore.SetParamSet(ctx, &initialParams)

	err := v3.MigrateStore(
		ctx,
		paramstore,
		storeKey,
		cdc,
	)
	require.NoError(t, err)

	// Get all the new parameters from the kvStore
	paramsBz := kvStore.Get(types.KeyPrefixParams)
	var migratedParams types.Params
	cdc.MustUnmarshal(paramsBz, &migratedParams)

	legacytestutil.AssertParamsEqual(t, initialParams, migratedParams)
}

func TestMigrate_Mainnet(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})
	kvStore := ctx.KVStore(storeKey)

	initialChainConfig := legacytypes.DefaultChainConfig()
	initialChainConfig.LondonBlock = nil
	initialChainConfig.ArrowGlacierBlock = nil
	initialChainConfig.MergeForkBlock = nil

	initialParams := legacytypes.LegacyParams{
		EvmDenom:     "akava",
		EnableCreate: true,
		EnableCall:   true,
		ExtraEIPs:    nil,
		ChainConfig:  initialChainConfig,
		// Start with a subset of allowed messages
		EIP712AllowedMsgs: legacytestutil.TestEIP712AllowedMsgs,
	}

	paramstore := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	).WithKeyTable(legacytypes.ParamKeyTable())

	paramstore.SetParamSet(ctx, &initialParams)

	err := v3.MigrateStore(
		ctx,
		paramstore,
		storeKey,
		cdc,
	)
	require.NoError(t, err)

	// Get all the new parameters from the kvStore
	paramsBz := kvStore.Get(types.KeyPrefixParams)
	var migratedParams types.Params
	cdc.MustUnmarshal(paramsBz, &migratedParams)

	// ensure migrated params match initial params
	legacytestutil.AssertParamsEqual(t, initialParams, migratedParams)
}

func TestKeyTableCompatiabilityWithKeeper(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})

	ak := app.Setup(false, nil).AccountKeeper

	// only used to set initial params
	initialSubspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	).WithKeyTable(legacytypes.ParamKeyTable())
	initialParams := legacytypes.DefaultParams()
	initialSubspace.SetParamSet(ctx, &initialParams)

	// vanilla subspace (no key table) that keeper
	// will register a key table on
	subspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	)
	keeper.NewKeeper(
		cdc, storeKey, tKey, authtypes.NewModuleAddress("gov"),
		ak,
		nil, nil, nil,
		vm.NewEVM,
		"",
		subspace,
	)

	// ensure that the migration is compatible with the keeper legacy
	// key table registration
	require.NotPanics(t, func() {
		v3.MigrateStore(
			ctx,
			subspace,
			storeKey,
			cdc,
		)
	}, "type mismatch with registered table")
}

func TestMigrationRegistersItsOwnKeyTable(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	paramStoreKey := sdk.NewKVStoreKey(paramtypes.ModuleName)
	paramStoreTKey := sdk.NewTransientStoreKey(paramtypes.TStoreKey)
	ctx := legacytestutil.NewDBContext([]storetypes.StoreKey{storeKey, paramStoreKey}, []storetypes.StoreKey{tKey, paramStoreTKey})

	// only used to set initial params
	initialSubspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	).WithKeyTable(legacytypes.ParamKeyTable())
	initialParams := legacytypes.DefaultParams()
	initialSubspace.SetParamSet(ctx, &initialParams)

	// vanilla subspace (no key table) that MigrateStore
	// will register a key table on
	subspace := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		paramStoreKey,
		paramStoreTKey,
		"evm",
	)
	// ensure that the migration is compatible with the keeper legacy
	// key table registration
	require.NotPanics(t, func() {
		v3.MigrateStore(
			ctx,
			subspace,
			storeKey,
			cdc,
		)
	})
}
