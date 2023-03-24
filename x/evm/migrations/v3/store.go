package v3

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v2types "github.com/evmos/ethermint/x/evm/migrations/v2/types"
	"github.com/evmos/ethermint/x/evm/types"
)

// MigrateStore runs the state migrations that includes upstream consensus
// versions v2 to v5. Kava consensus version diverges from upstream at v2.
func MigrateStore(
	ctx sdk.Context,
	paramstore types.Subspace,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var legacyParams v2types.V2Params
	paramstore.GetParamSetIfExists(ctx, &legacyParams)

	// -------------------------------------------------------------------------
	// Upstream v2 to v3
	// New GrayGlacierBlock and MergeNetsplitBlock in ChainConfig parameter.
	// Any new fields are disabled / nil. These should be nil if we leave them
	// out because of the default value, but set explicitly here to nil.

	newChainConfig := types.ChainConfig{
		HomesteadBlock:      legacyParams.ChainConfig.HomesteadBlock,
		DAOForkBlock:        legacyParams.ChainConfig.DAOForkBlock,
		DAOForkSupport:      legacyParams.ChainConfig.DAOForkSupport,
		EIP150Block:         legacyParams.ChainConfig.EIP150Block,
		EIP150Hash:          legacyParams.ChainConfig.EIP150Hash,
		EIP155Block:         legacyParams.ChainConfig.EIP155Block,
		EIP158Block:         legacyParams.ChainConfig.EIP158Block,
		ByzantiumBlock:      legacyParams.ChainConfig.ByzantiumBlock,
		ConstantinopleBlock: legacyParams.ChainConfig.ConstantinopleBlock,
		PetersburgBlock:     legacyParams.ChainConfig.PetersburgBlock,
		IstanbulBlock:       legacyParams.ChainConfig.IstanbulBlock,
		MuirGlacierBlock:    legacyParams.ChainConfig.MuirGlacierBlock,
		BerlinBlock:         legacyParams.ChainConfig.BerlinBlock,
		LondonBlock:         legacyParams.ChainConfig.LondonBlock,
		ArrowGlacierBlock:   legacyParams.ChainConfig.ArrowGlacierBlock,

		// This is an old field, but renamed from mergeForkBlock
		MergeNetsplitBlock: legacyParams.ChainConfig.MergeForkBlock,

		// New fields are nil
		GrayGlacierBlock: nil,
		ShanghaiBlock:    nil,
		CancunBlock:      nil,
	}

	// -------------------------------------------------------------------------
	// Upstream v3 to v4
	// Moves from deprecated Cosmos SDK params module to x/evm module state.

	// Params in store is currently empty
	store := ctx.KVStore(storeKey)

	newParams := types.Params{
		EvmDenom:            legacyParams.EvmDenom,
		EnableCreate:        legacyParams.EnableCreate,
		EnableCall:          legacyParams.EnableCall,
		ExtraEIPs:           legacyParams.ExtraEIPs,
		ChainConfig:         newChainConfig,
		EIP712AllowedMsgs:   MigrateEIP712AllowedMsgs(legacyParams.EIP712AllowedMsgs),
		AllowUnprotectedTxs: false, // Upstream v1 to v2
	}

	if err := newParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&newParams)
	store.Set(types.KeyPrefixParams, bz)

	return nil
}

// MigrateEIP712AllowedMsgs converts the old EIP712AllowedMsgs to the new one.
// No changes, just a type conversion.
func MigrateEIP712AllowedMsgs(old []v2types.V2EIP712AllowedMsg) []types.EIP712AllowedMsg {
	new := make([]types.EIP712AllowedMsg, len(old))
	for i, msg := range old {
		new[i] = types.EIP712AllowedMsg{
			MsgTypeUrl:       msg.MsgTypeUrl,
			MsgValueTypeName: msg.MsgValueTypeName,
			ValueTypes:       MigrateEIP712MsgAttrTypes(msg.ValueTypes),
			NestedTypes:      MigrateNestedTypes(msg.NestedTypes),
		}
	}

	return new
}

// MigrateEIP712MsgAttrTypes converts the old EIP712MsgAttrTypes to the new one.
// No changes, just a type conversion.
func MigrateEIP712MsgAttrTypes(old []v2types.V2EIP712MsgAttrType) []types.EIP712MsgAttrType {
	new := make([]types.EIP712MsgAttrType, len(old))
	for i, msg := range old {
		// We can directly assign because of the same fields
		new[i] = types.EIP712MsgAttrType(msg)
	}

	return new
}

// MigrateNestedTypes converts the old EIP712NestedMsgTypes to the new one.
// No changes, just a type conversion.
func MigrateNestedTypes(old []v2types.V2EIP712NestedMsgType) []types.EIP712NestedMsgType {
	new := make([]types.EIP712NestedMsgType, len(old))
	for i, msg := range old {
		new[i] = types.EIP712NestedMsgType{
			Name:  msg.Name,
			Attrs: MigrateEIP712MsgAttrTypes(msg.Attrs),
		}
	}

	return new
}
