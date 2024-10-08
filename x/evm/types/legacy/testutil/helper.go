package testutil

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cometbft/cometbft-db"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/ethermint/x/evm/types"
	legacytypes "github.com/evmos/ethermint/x/evm/types/legacy"
	"github.com/stretchr/testify/require"
)

// NewDefaultContext with multile mounted stores
func NewDBContext(keys []storetypes.StoreKey, tkeys []storetypes.StoreKey) sdk.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db)

	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}

	for _, tkey := range tkeys {
		cms.MountStoreWithDB(tkey, storetypes.StoreTypeTransient, db)
	}

	err := cms.LoadLatestVersion()
	if err != nil {
		panic(err)
	}

	return sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())
}

func AssertParamsEqual(t *testing.T, legacyParams legacytypes.LegacyParams, params types.Params) {
	//
	// Check Primitive Top Level Values
	//
	require.Equal(t, legacyParams.EvmDenom, params.EvmDenom)
	require.Equal(t, legacyParams.EnableCall, params.EnableCall)
	require.Equal(t, legacyParams.EnableCreate, params.EnableCreate)
	require.Equal(t, legacyParams.ExtraEIPs, params.ExtraEIPs)

	//
	// Check Chain Config
	//
	legacyChainConfig := legacyParams.ChainConfig
	chainConfig := params.ChainConfig

	require.Equal(t, legacyChainConfig.HomesteadBlock, chainConfig.HomesteadBlock)
	require.Equal(t, legacyChainConfig.DAOForkBlock, chainConfig.DAOForkBlock)
	require.Equal(t, legacyChainConfig.DAOForkSupport, chainConfig.DAOForkSupport)
	require.Equal(t, legacyChainConfig.EIP150Block, chainConfig.EIP150Block)
	require.Equal(t, legacyChainConfig.EIP150Hash, chainConfig.EIP150Hash)
	require.Equal(t, legacyChainConfig.ByzantiumBlock, chainConfig.ByzantiumBlock)
	require.Equal(t, legacyChainConfig.ConstantinopleBlock, chainConfig.ConstantinopleBlock)
	require.Equal(t, legacyChainConfig.PetersburgBlock, chainConfig.PetersburgBlock)
	require.Equal(t, legacyChainConfig.IstanbulBlock, chainConfig.IstanbulBlock)
	require.Equal(t, legacyChainConfig.MuirGlacierBlock, chainConfig.MuirGlacierBlock)
	require.Equal(t, legacyChainConfig.BerlinBlock, chainConfig.BerlinBlock)
	require.Equal(t, legacyChainConfig.LondonBlock, chainConfig.LondonBlock)
	require.Equal(t, legacyChainConfig.ArrowGlacierBlock, chainConfig.ArrowGlacierBlock)
	// renamed value
	require.Equal(t, legacyChainConfig.MergeForkBlock, chainConfig.MergeNetsplitBlock)
	// new values that should be nil
	require.Nil(t, chainConfig.GrayGlacierBlock)
	require.Nil(t, chainConfig.ShanghaiBlock)
	require.Nil(t, chainConfig.CancunBlock)

	//
	// EIP712
	//
	require.Equal(t, legacyParams.EIP712AllowedMsgs, params.EIP712AllowedMsgs)

	//
	// New Parameter
	//
	require.Equal(t, false, params.AllowUnprotectedTxs)
}

var TestEIP712AllowedMsgs = []types.EIP712AllowedMsg{
	{
		MsgTypeUrl:       "/kava.evmutil.v1beta1.MsgConvertERC20ToCoin",
		MsgValueTypeName: "MsgValueEVMConvertERC20ToCoin",
		ValueTypes: []types.EIP712MsgAttrType{
			{Name: "initiator", Type: "string"},
			{Name: "receiver", Type: "string"},
			{Name: "kava_erc20_address", Type: "string"},
			{Name: "amount", Type: "string"},
		},
	},
	{
		MsgTypeUrl:       "/kava.evmutil.v1beta1.MsgConvertCoinToERC20",
		MsgValueTypeName: "MsgValueEVMConvertCoinToERC20",
		ValueTypes: []types.EIP712MsgAttrType{
			{Name: "initiator", Type: "string"},
			{Name: "receiver", Type: "string"},
			{Name: "amount", Type: "Coin"},
		},
	},
	// x/earn
	{
		MsgTypeUrl:       "/kava.earn.v1beta1.MsgDeposit",
		MsgValueTypeName: "MsgValueEarnDeposit",
		ValueTypes: []types.EIP712MsgAttrType{
			{Name: "depositor", Type: "string"},
			{Name: "amount", Type: "Coin"},
			{Name: "strategy", Type: "int32"},
		},
	},
	{
		MsgTypeUrl:       "/kava.earn.v1beta1.MsgWithdraw",
		MsgValueTypeName: "MsgValueEarnWithdraw",
		ValueTypes: []types.EIP712MsgAttrType{
			{Name: "from", Type: "string"},
			{Name: "amount", Type: "Coin"},
			{Name: "strategy", Type: "int32"},
		},
	},
}
