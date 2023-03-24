package v3_test

import (
	"encoding/json"
	"testing"

	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	v2types "github.com/evmos/ethermint/x/evm/migrations/v2/types"
	v3 "github.com/evmos/ethermint/x/evm/migrations/v3"
)

func TestMigrate(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	ctx := testutil.DefaultContext(storeKey, tKey)
	kvStore := ctx.KVStore(storeKey)

	paramstore := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		storeKey,
		tKey,
		"evm",
	).WithKeyTable(v2types.ParamKeyTable())

	initialParams := v2types.DefaultParams()
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

	// No changes to existing params
	require.Equal(t, initialParams.EvmDenom, migratedParams.EvmDenom)
	require.Equal(t, initialParams.EnableCall, migratedParams.EnableCall)
	require.Equal(t, initialParams.EnableCreate, migratedParams.EnableCreate)
	require.Equal(t, initialParams.ExtraEIPs, migratedParams.ExtraEIPs)
	require.ElementsMatch(t, initialParams.EIP712AllowedMsgs, migratedParams.EIP712AllowedMsgs)

	// New param should be false
	require.Equal(t, false, migratedParams.AllowUnprotectedTxs)

	// New ChainConfig options are set to nil
	expectedChainConfig := types.DefaultChainConfig()
	expectedChainConfig.GrayGlacierBlock = nil
	expectedChainConfig.ShanghaiBlock = nil
	expectedChainConfig.CancunBlock = nil

	require.EqualValues(t, expectedChainConfig, migratedParams.ChainConfig)
}

func TestMigrate_Mainnet(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey(types.TransientKey)
	ctx := testutil.DefaultContext(storeKey, tKey)
	kvStore := ctx.KVStore(storeKey)

	initialChainConfig := v2types.DefaultChainConfig()
	initialChainConfig.LondonBlock = nil
	initialChainConfig.ArrowGlacierBlock = nil
	initialChainConfig.MergeForkBlock = nil

	initialParams := v2types.V2Params{
		EvmDenom:     "akava",
		EnableCreate: true,
		EnableCall:   true,
		ExtraEIPs:    nil,
		ChainConfig:  initialChainConfig,
		// Start with a subset of allowed messages
		EIP712AllowedMsgs: []v2types.V2EIP712AllowedMsg{
			{
				MsgTypeUrl:       "/kava.evmutil.v1beta1.MsgConvertERC20ToCoin",
				MsgValueTypeName: "MsgValueEVMConvertERC20ToCoin",
				ValueTypes: []v2types.V2EIP712MsgAttrType{
					{Name: "initiator", Type: "string"},
					{Name: "receiver", Type: "string"},
					{Name: "kava_erc20_address", Type: "string"},
					{Name: "amount", Type: "string"},
				},
			},
			{
				MsgTypeUrl:       "/kava.evmutil.v1beta1.MsgConvertCoinToERC20",
				MsgValueTypeName: "MsgValueEVMConvertCoinToERC20",
				ValueTypes: []v2types.V2EIP712MsgAttrType{
					{Name: "initiator", Type: "string"},
					{Name: "receiver", Type: "string"},
					{Name: "amount", Type: "Coin"},
				},
			},
			// x/earn
			{
				MsgTypeUrl:       "/kava.earn.v1beta1.MsgDeposit",
				MsgValueTypeName: "MsgValueEarnDeposit",
				ValueTypes: []v2types.V2EIP712MsgAttrType{
					{Name: "depositor", Type: "string"},
					{Name: "amount", Type: "Coin"},
					{Name: "strategy", Type: "int32"},
				},
			},
			{
				MsgTypeUrl:       "/kava.earn.v1beta1.MsgWithdraw",
				MsgValueTypeName: "MsgValueEarnWithdraw",
				ValueTypes: []v2types.V2EIP712MsgAttrType{
					{Name: "from", Type: "string"},
					{Name: "amount", Type: "Coin"},
					{Name: "strategy", Type: "int32"},
				},
			},
		},
	}

	paramstore := paramtypes.NewSubspace(
		cdc,
		encCfg.Amino,
		storeKey,
		tKey,
		"evm",
	).WithKeyTable(v2types.ParamKeyTable())

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

	require.Equal(t, initialParams.EvmDenom, migratedParams.EvmDenom)
	require.Equal(t, initialParams.EnableCall, migratedParams.EnableCall)
	require.Equal(t, initialParams.EnableCreate, migratedParams.EnableCreate)
	require.Equal(t, false, migratedParams.AllowUnprotectedTxs)
	require.Equal(t, initialParams.ExtraEIPs, migratedParams.ExtraEIPs)

	expectedEIP712AllowedMsgsJson, err := json.Marshal(initialParams.EIP712AllowedMsgs)
	require.NoError(t, err)

	migratedEIP712AllowedMsgsJson, err := json.Marshal(migratedParams.EIP712AllowedMsgs)
	require.NoError(t, err)

	// Convert to JSON since they are different types but of same field and values
	require.JSONEq(t, string(expectedEIP712AllowedMsgsJson), string(migratedEIP712AllowedMsgsJson))

	expectedChainConfig := types.DefaultChainConfig()
	// Previously nil ChainConfig options are still nil
	expectedChainConfig.LondonBlock = nil
	expectedChainConfig.ArrowGlacierBlock = nil
	expectedChainConfig.MergeNetsplitBlock = nil

	// New ChainConfig options are set to nil
	expectedChainConfig.GrayGlacierBlock = nil
	expectedChainConfig.ShanghaiBlock = nil
	expectedChainConfig.CancunBlock = nil

	require.EqualValues(t, expectedChainConfig, migratedParams.ChainConfig)
}
